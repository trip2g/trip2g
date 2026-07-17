package graph

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"

	"trip2g/cmd/fleet/internal/fleet"
)

// gqlDoerFunc adapts a func to genqlient's graphql.Doer.
type gqlDoerFunc func(*http.Request) (*http.Response, error)

func (f gqlDoerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

// fakeGQL builds a genqlient client backed by respond, keyed by operationName.
// ListCronWebhooks returns an empty list when respond returns an error for it.
func fakeGQL(respond func(op string, vars json.RawMessage) (string, error)) graphql.Client {
	inner := respond
	respond = func(op string, vars json.RawMessage) (string, error) {
		data, err := inner(op, vars)
		if err != nil && op == "ListCronWebhooks" {
			return `{"admin":{"allCronWebhooks":{"nodes":[]}}}`, nil
		}
		return data, err
	}
	doer := gqlDoerFunc(func(req *http.Request) (*http.Response, error) {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		var in struct {
			OperationName string          `json:"operationName"`
			Variables     json.RawMessage `json:"variables"`
		}
		if uerr := json.Unmarshal(raw, &in); uerr != nil {
			return nil, uerr
		}
		data, rerr := respond(in.OperationName, in.Variables)
		var env string
		if rerr != nil {
			env = `{"errors":[{"message":` + strconv.Quote(rerr.Error()) + `}]}`
		} else {
			env = `{"data":` + data + `}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(env)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
	return graphql.NewClient("http://fake.local/_system/graphql", doer)
}

const graphDiscoverData = `{"notePaths":[
  {"value":"roles/indexer.md","content":"index the wiki","latestNoteView":{"meta":[
    {"key":"mode","raw":"change"},
    {"key":"trigger_on","raw":"[create update]"},
    {"key":"trigger_include","raw":"[wiki/**]"},
    {"key":"write_patterns","raw":"[index/**]"}
  ]}},
  {"value":"roles/writer.md","content":"write wiki notes","latestNoteView":{"meta":[
    {"key":"mode","raw":"change"},
    {"key":"trigger_on","raw":"[create update]"},
    {"key":"trigger_include","raw":"[inbox/**]"},
    {"key":"write_patterns","raw":"[wiki/**]"},
    {"key":"tools","raw":"[not_offered_tool]"}
  ]}}
]}`

const graphWebhooksData = `{"admin":{"allChangeWebhooks":{"nodes":[
  {"id":1,"description":"fleet:f1:roles/writer.md#abc123"},
  {"id":2,"description":"fleet:f2:roles/writer.md#abc123"},
  {"id":3,"description":"fleet:f1:roles/ghost.md#zzz"},
  {"id":4,"description":"unrelated human webhook"}
]}}}`

func newTestServer() *Server {
	gql := fakeGQL(func(op string, _ json.RawMessage) (string, error) {
		switch op {
		case "DiscoverRoles":
			return graphDiscoverData, nil
		case "ListChangeWebhooks":
			return graphWebhooksData, nil
		}
		return "", errors.New("unexpected op " + op)
	})
	cfg := fleet.Config{
		FleetID:       "f1",
		AgentsFolder:  "roles/",
		OfferedTools:  []string{"read_note", "write_note"},
		Trip2gBaseURL: "http://hub.local:20081",
	}
	return NewServer(fleet.NewDiscovery(gql, cfg.FleetID, cfg.AgentsFolder, cfg.OfferedTools), gql, cfg)
}

func TestServerJSON(t *testing.T) {
	s := newTestServer()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/graph.json", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var g Graph
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &g))

	// 2 discovered roles + 1 ghost marker.
	require.Len(t, g.Nodes, 3)
	byPath := map[string]Node{}
	for _, n := range g.Nodes {
		byPath[n.NotePath] = n
	}

	// writer -> indexer trigger edge (writes wiki/**, indexer triggers on wiki/**).
	require.Len(t, g.Edges, 1)
	require.Equal(t, "roles/writer.md", g.Edges[0].From)
	require.Equal(t, "roles/indexer.md", g.Edges[0].To)
	require.Equal(t, "trigger", g.Edges[0].Kind)

	// writer claims a tool the fleet does not offer -> invalid but kept.
	require.False(t, byPath["roles/writer.md"].Valid)
	require.True(t, byPath["roles/indexer.md"].Valid)

	// Registry-derived state.
	require.True(t, byPath["roles/writer.md"].Registered)
	require.Equal(t, []string{"f1", "f2"}, byPath["roles/writer.md"].FleetIDs)
	require.True(t, byPath["roles/ghost.md"].Ghost)
	require.False(t, byPath["roles/indexer.md"].Registered)

	kinds := map[string]bool{}
	for _, f := range g.Findings {
		kinds[f.Kind] = true
	}
	require.True(t, kinds["conflict"], "expected conflict finding")
	require.True(t, kinds["drift"], "expected drift finding")
	require.True(t, kinds["invalid-role"], "expected invalid-role finding")
}

func TestServerUI(t *testing.T) {
	s := newTestServer()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	require.Contains(t, rec.Body.String(), "graph.json")
}

func TestServerUIMermaidURL(t *testing.T) {
	s := newTestServer()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, 200, rec.Code)
	// Mermaid must be loaded from the hub, not from the fleet server itself.
	require.Contains(t, rec.Body.String(), "http://hub.local:20081/assets/mermaid.min.js")
}

func TestParseFleetMarker(t *testing.T) {
	tests := []struct {
		desc string
		want Marker
		ok   bool
	}{
		{"fleet:f1:roles/a.md#abc", Marker{FleetID: "f1", NotePath: "roles/a.md"}, true},
		{"fleetcron:f2:roles/b.md#def", Marker{FleetID: "f2", NotePath: "roles/b.md", Cron: true}, true},
		{"fleet:f1:roles/no-ver.md", Marker{FleetID: "f1", NotePath: "roles/no-ver.md"}, true},
		{"human webhook for slack", Marker{}, false},
		{"fleet:brokenmarker", Marker{}, false},
	}
	for _, tt := range tests {
		got, ok := ParseFleetMarker(tt.desc)
		require.Equal(t, tt.ok, ok, tt.desc)
		require.Equal(t, tt.want, got, tt.desc)
	}
}
