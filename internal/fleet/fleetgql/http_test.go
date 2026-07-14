package fleetgql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/fleet"
)

// fakeSource is a static RoleSource for handler tests.
type fakeSource struct{ roles []fleet.Role }

func (f fakeSource) DiscoverParsed(context.Context) ([]fleet.Role, []error) { return f.roles, nil }

// post runs a GraphQL query against the handler and returns the decoded body.
func post(t *testing.T, h http.Handler, query string) map[string]json.RawMessage {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"query": query})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code, rec.Body.String())

	var env struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors json.RawMessage            `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Nil(t, env.Errors, "graphql errors: %s", env.Errors)
	return env.Data
}

func TestHandlerRoles(t *testing.T) {
	src := fakeSource{roles: []fleet.Role{
		{
			NotePath: "roles/indexer.md", Executor: "code", Model: "gpt-4o",
			TriggerOn: []string{"create", "update"}, Mode: "change",
			TriggerInclude: []string{"wiki/**"}, WritePatterns: []string{"index/**"},
		},
		{NotePath: "roles/writer.md"}, // executor "" -> default "llm"; model "" -> null
	}}
	h := NewHTTPHandler(src, nil)

	data := post(t, h, `{ roles { name path executor model triggerInclude writePatterns } }`)
	var got struct {
		Roles []struct {
			Name, Path, Executor string
			Model                *string
			TriggerInclude       []string
			WritePatterns        []string
		}
	}
	require.NoError(t, json.Unmarshal([]byte(`{"Roles":`+string(data["roles"])+`}`), &got))
	require.Len(t, got.Roles, 2)

	require.Equal(t, "indexer", got.Roles[0].Name)
	require.Equal(t, "roles/indexer.md", got.Roles[0].Path)
	require.Equal(t, "code", got.Roles[0].Executor)
	require.NotNil(t, got.Roles[0].Model)
	require.Equal(t, "gpt-4o", *got.Roles[0].Model)

	require.Equal(t, "llm", got.Roles[1].Executor) // default projection
	require.Nil(t, got.Roles[1].Model)             // empty model -> null
}

func TestHandlerRoleGraph(t *testing.T) {
	src := fakeSource{roles: []fleet.Role{
		rgChange("roles/writer.md", []string{"inbox/**"}, []string{"wiki/**"}),
		rgChange("roles/indexer.md", []string{"wiki/**"}, []string{"index/**"}),
	}}
	h := NewHTTPHandler(src, nil)

	data := post(t, h, `{ roleGraph {
		nodes { role inboxGlob outboxGlob orphan }
		edges { from to kind exact cutByDepth }
		cycles
	} }`)
	var got struct {
		RoleGraph struct {
			Nodes []struct {
				Role                  string
				InboxGlob, OutboxGlob []string
				Orphan                bool
			}
			Edges []struct {
				From, To, Kind string
				Exact          bool
				CutByDepth     bool
			}
			Cycles [][]string
		}
	}
	require.NoError(t, json.Unmarshal([]byte(`{"RoleGraph":`+string(data["roleGraph"])+`}`), &got))

	require.Len(t, got.RoleGraph.Nodes, 2)
	require.Len(t, got.RoleGraph.Edges, 1)
	e := got.RoleGraph.Edges[0]
	require.Equal(t, "roles/writer.md", e.From)
	require.Equal(t, "roles/indexer.md", e.To)
	require.Equal(t, "TRIGGER", e.Kind) // enum marshals to uppercase wire value
	require.True(t, e.Exact)
	require.False(t, e.CutByDepth)
	require.Empty(t, got.RoleGraph.Cycles)
}

// rgChange builds a change-triggerable role for handler tests.
func rgChange(path string, include, write []string) fleet.Role {
	return fleet.Role{
		NotePath: path, Mode: "change", TriggerOn: []string{"create", "update"},
		TriggerInclude: include, WritePatterns: write,
	}
}
