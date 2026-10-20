package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/mcp"
	"trip2g/internal/features"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"
)

var agentFieldNames = []string{"short_query", "rephrased_query", "expected_answer"} //nolint:gochecknoglobals // test fixture

func TestSearchPassesShortQueryToTheTextLane(t *testing.T) {
	var feats features.Features
	env := searchRerankEnv(t, feats, "http://unused")
	noteA := env.LiveNoteViewsFunc().PathMap["a.md"]
	env.SearchLiveNotesFunc = func(query string, _, _ int) ([]appmodel.SearchResult, error) {
		if query != "pin tab" {
			return nil, nil
		}
		return []appmodel.SearchResult{{NoteView: noteA, URL: noteA.Permalink}}, nil
	}

	payload := rerankSearchCallArgs(t, env,
		`{"query":"how do I pin a tab","short_query":"pin tab","rephrased_query":"r","expected_answer":"e"}`)

	require.Len(t, payload.Results, 1)
	require.Equal(t, "a.md", payload.Results[0].NotePath)
	var queries []string
	for _, c := range env.SearchLiveNotesCalls() {
		queries = append(queries, c.Query)
	}
	require.Equal(t, []string{"how do I pin a tab", "pin tab"}, queries)
	require.Empty(t, env.OpenAICalls(), "vector fields do nothing with vector search off")
}

func TestFederatedSearchForwardsAgentFields(t *testing.T) {
	const args = `"query":"q","short_query":"s","rephrased_query":"r","expected_answer":"e"`
	want := appmodel.MCPSearchParams{Query: "q", ShortQuery: "s", RephrasedQuery: "r", ExpectedAnswer: "e"}

	tests := []struct {
		name  string
		kbID  string
		wantK string
	}{
		{name: "direct peer", kbID: "bob"},
		{name: "nested base", kbID: "bob/deep", wantK: "deep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kbNote := &appmodel.NoteView{
				PathID:             17,
				MCPFederationKBURL: "https://bob.example/_system/mcp",
				MCPFederationKBID:  "bob",
			}
			nvs := appmodel.NewNoteViews()
			nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

			var got appmodel.MCPSearchParams
			capture := func(_ context.Context, params appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
				got = params
				return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: "ok"}}}, nil
			}
			federation := &federationMock{searchFunc: capture, federatedSearchFunc: capture}
			env := &EnvMock{
				FeaturesFunc:               func() features.Features { return features.Features{} },
				FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
				MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
				LatestNoteViewsFunc:        func() *appmodel.NoteViews { return nvs },
				CanReadNoteFunc: func(context.Context, *appmodel.NoteView) (bool, error) {
					return true, nil
				},
				FederationClientFunc: func(context.Context, string) (appmodel.Federation, error) {
					return federation, nil
				},
				FederationMaxDepthFunc: func() int { return 3 },
			}

			params := mcp.CallToolParams{
				Name:      "federated_search",
				Arguments: json.RawMessage(`{` + args + `,"kb_id":"` + tt.kbID + `"}`),
			}
			paramsJSON, err := json.Marshal(params)
			require.NoError(t, err)
			resp := callMCP(t, env, mcp.Request{JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1})
			require.Nil(t, resp.Error)

			want.KBID = tt.wantK
			require.Equal(t, want, got)
		})
	}
}

// The peer's side of forwarding: an older peer that has no such fields must
// still decode the call, and the wire form must carry them by name.
func TestAgentFieldsWireForm(t *testing.T) {
	b, err := json.Marshal(appmodel.MCPSearchParams{Query: "q", ShortQuery: "s", RephrasedQuery: "r", ExpectedAnswer: "e"})
	require.NoError(t, err)
	require.JSONEq(t, `{"query":"q","short_query":"s","rephrased_query":"r","expected_answer":"e"}`, string(b))

	b, err = json.Marshal(appmodel.MCPSearchParams{Query: "q"})
	require.NoError(t, err)
	require.JSONEq(t, `{"query":"q"}`, string(b), "unset fields stay off the wire")

	var older struct {
		Query string `json:"query"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"query":"q","short_query":"s"}`), &older))
}

func TestToolsListDescribesAgentFieldsInTheBaseLanguage(t *testing.T) {
	tests := []struct {
		name     string
		langs    []string
		wantLang string
	}{
		{name: "most notes are Russian", langs: []string{"ru", "en", "ru", ""}, wantLang: " in Russian"},
		{name: "a tie picks the first code", langs: []string{"ru", "en"}, wantLang: " in English"},
		{name: "no note has a language", langs: []string{"", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var feats features.Features
			env := searchRerankEnv(t, feats, "http://unused")
			nvs := appmodel.NewNoteViews()
			for i, lang := range tt.langs {
				nvs.List = append(nvs.List, &appmodel.NoteView{Path: strings.Repeat("n", i+1) + ".md", Lang: lang})
			}
			env.LatestNoteViewsFunc = func() *appmodel.NoteViews { return nvs }

			for name, tool := range listSearchTools(t, env) {
				for _, field := range agentFieldNames {
					prop, ok := tool.InputSchema.Properties[field]
					require.True(t, ok, "%s must advertise %s", name, field)
					require.Equal(t, "string", prop.Type)
					require.True(t, strings.HasPrefix(prop.Description, "Optional."), prop.Description)
					if tt.wantLang == "" {
						require.NotContains(t, prop.Description, " in English")
						require.NotContains(t, prop.Description, " in Russian")
						continue
					}
					require.Contains(t, prop.Description, tt.wantLang)
				}
				require.NotContains(t, tool.InputSchema.Required, "short_query")
			}
		})
	}
}
