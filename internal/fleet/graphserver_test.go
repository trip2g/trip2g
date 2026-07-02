package fleet

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/fleet/graph"
)

// graphDiscoverData fakes the admin lane with two roles (writer triggers
// indexer); the writer also claims a tool the fleet does not offer.
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

func newTestGraphServer() *GraphServer {
	gql := fakeAdminGQL(func(op string, _ json.RawMessage) (string, error) {
		switch op {
		case "DiscoverRoles":
			return graphDiscoverData, nil
		case "ListChangeWebhooks":
			return graphWebhooksData, nil
		}
		return "", errors.New("unexpected op " + op)
	})
	cfg := Config{FleetID: "f1", AgentsFolder: "roles/", OfferedTools: []string{"read_note", "write_note"}}
	return NewGraphServer(NewDiscovery(gql, cfg.AgentsFolder, cfg.OfferedTools), gql, cfg)
}

func TestGraphServerJSON(t *testing.T) {
	s := newTestGraphServer()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/graph.json", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var g graph.Graph
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &g))

	// 2 discovered roles + 1 ghost marker.
	require.Len(t, g.Nodes, 3)
	byPath := map[string]graph.Node{}
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

	// Registry-derived state: writer registered + conflicting f2 claim; ghost kept.
	require.True(t, byPath["roles/writer.md"].Registered)
	require.Equal(t, []string{"f1", "f2"}, byPath["roles/writer.md"].FleetIDs)
	require.True(t, byPath["roles/ghost.md"].Ghost)
	require.False(t, byPath["roles/indexer.md"].Registered)

	kinds := map[string]bool{}
	for _, f := range g.Findings {
		kinds[f.Kind] = true
	}
	require.True(t, kinds["conflict"], "expected conflict finding, got %+v", g.Findings)
	require.True(t, kinds["drift"], "expected drift finding, got %+v", g.Findings)
	require.True(t, kinds["invalid-role"], "expected invalid-role finding, got %+v", g.Findings)
}

func TestGraphServerUI(t *testing.T) {
	s := newTestGraphServer()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	require.Contains(t, rec.Body.String(), "graph.json")
}

func TestParseFleetMarker(t *testing.T) {
	tests := []struct {
		desc string
		want graph.Marker
		ok   bool
	}{
		{"fleet:f1:roles/a.md#abc", graph.Marker{FleetID: "f1", NotePath: "roles/a.md"}, true},
		{"fleetcron:f2:roles/b.md#def", graph.Marker{FleetID: "f2", NotePath: "roles/b.md", Cron: true}, true},
		{"fleet:f1:roles/no-ver.md", graph.Marker{FleetID: "f1", NotePath: "roles/no-ver.md"}, true},
		{"human webhook for slack", graph.Marker{}, false},
		{"fleet:brokenmarker", graph.Marker{}, false},
	}
	for _, tt := range tests {
		got, ok := parseFleetMarker(tt.desc)
		require.Equal(t, tt.ok, ok, tt.desc)
		require.Equal(t, tt.want, got, tt.desc)
	}
}
