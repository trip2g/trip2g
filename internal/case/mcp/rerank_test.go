package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/mcp"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"
	"trip2g/internal/openai"
)

// embeddingServer returns an OpenAI-compatible /v1/embeddings server that
// always returns the given query vector.
func embeddingServer(t *testing.T, vector []float32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "index": 0, "embedding": vector},
			},
			"model": "test-model",
			"usage": map[string]int{"prompt_tokens": 1, "total_tokens": 1},
		}))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// rerankServer scores each doc via scoreFor (TEI wire: bare [{index,score}])
// and counts requests so tests can assert the reranker was (not) called.
func rerankServer(t *testing.T, calls *atomic.Int64, scoreFor func(doc string) float64) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req struct {
			Texts []string `json:"texts"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		type result struct {
			Index int     `json:"index"`
			Score float64 `json:"score"`
		}
		var out []result
		for i, d := range req.Texts {
			out = append(out, result{Index: i, Score: scoreFor(d)})
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(out))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// searchRerankEnv builds an Env where both text and vector lanes rank A above
// B, so only a reranker can put B first.
func searchRerankEnv(t *testing.T, feats features.Features, embURL string) *EnvMock {
	t.Helper()
	noteA := &appmodel.NoteView{Path: "a.md", PathID: 1, Title: "A", Permalink: "/a"}
	noteB := &appmodel.NoteView{Path: "b.md", PathID: 2, Title: "B", Permalink: "/b"}

	return &EnvMock{
		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		SearchLiveNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: noteA, URL: noteA.Permalink, Score: 2.0, HighlightedContent: []string{"alpha"}},
				{NoteView: noteB, URL: noteB.Permalink, Score: 1.0, HighlightedContent: []string{"beta"}},
			}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return []appmodel.NoteChunk{
				{NotePath: "a.md", ChunkIndex: 0, Content: "A > H\n\nalpha passage", Embedding: []float32{1, 0}},
				{NotePath: "b.md", ChunkIndex: 0, Content: "B > H\n\nbeta passage", Embedding: []float32{0.9, 0.436}},
			}
		},
		LiveNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{
				List:    []*appmodel.NoteView{noteA, noteB},
				PathMap: map[string]*appmodel.NoteView{"a.md": noteA, "b.md": noteB},
			}
		},
		FeaturesFunc: func() features.Features { return feats },
		OpenAIFunc:   func() *openai.Client { return openai.New("test-key", "test-model", embURL+"/v1") },
		PublicURLFunc: func() string {
			return "http://localhost"
		},
		NoteURLFunc: func(note *appmodel.NoteView) string { return "http://localhost" + note.Permalink },
		LoggerFunc:  func() logger.Logger { return &logger.DummyLogger{} },
		CanReadNoteFunc: func(context.Context, *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
	}
}

func rerankSearchCall(t *testing.T, env *EnvMock) mcp.SearchResultPayload {
	t.Helper()
	return rerankSearchCallArgs(t, env, `{"query":"q"}`)
}

// rerankSearchCallArgs is the same call with the raw tool arguments spelled out,
// so a case can express the caller's rerank preference the way an agent does.
func rerankSearchCallArgs(t *testing.T, env *EnvMock, args string) mcp.SearchResultPayload {
	t.Helper()
	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(args),
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)
	resp := callMCP(t, env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.False(t, result.IsError, "content: %v", result.Content)
	return decodePayload[mcp.SearchResultPayload](t, result)
}

// rerankFeatures wires a configured cross-encoder at ceURL. byDefault decides
// whether a caller that says nothing gets reranked.
func rerankFeatures(ceURL string, byDefault bool) features.Features {
	var feats features.Features
	feats.VectorSearch.Enabled = true
	feats.VectorSearch.Reranker = features.RerankerConfig{
		Enabled:        true,
		Default:        byDefault,
		BaseURL:        ceURL,
		Model:          "test",
		TopN:           50,
		OutputK:        20,
		BlendWeight:    0.9,
		TimeoutSeconds: 5,
	}
	return feats
}

// ceFlipsToB scores so that the cross-encoder prefers B; with a high blend
// weight that is enough to flip a near-tie, which is how these cases tell
// "reranked" from "not reranked" by looking at the order alone.
func ceFlipsToB(t *testing.T, calls *atomic.Int64) string {
	t.Helper()
	return rerankServer(t, calls, func(doc string) float64 {
		if strings.Contains(doc, "beta") {
			return 0.99
		}
		return 0.01
	}).URL
}

func TestSearchRerankWhenAsked(t *testing.T) {
	// Configured but opt-in: the argument is what turns the second stage on.
	var ceCalls atomic.Int64
	ceURL := ceFlipsToB(t, &ceCalls)
	embSrv := embeddingServer(t, []float32{1, 0})

	env := searchRerankEnv(t, rerankFeatures(ceURL, false), embSrv.URL)
	payload := rerankSearchCallArgs(t, env, `{"query":"q","rerank":true}`)

	require.GreaterOrEqual(t, ceCalls.Load(), int64(1), "reranker must be called")
	require.Len(t, payload.Results, 2)
	require.Equal(t, "b.md", payload.Results[0].NotePath, "CE-preferred note must rank first")
}

func TestSearchDoesNotRerankUnlessAsked(t *testing.T) {
	// The sidecar is up and would reorder — silence must still cost nothing.
	var ceCalls atomic.Int64
	ceURL := ceFlipsToB(t, &ceCalls)
	embSrv := embeddingServer(t, []float32{1, 0})

	env := searchRerankEnv(t, rerankFeatures(ceURL, false), embSrv.URL)
	payload := rerankSearchCall(t, env)

	require.Equal(t, int64(0), ceCalls.Load(), "an unasked search must not pay for the cross-encoder")
	require.Equal(t, "a.md", payload.Results[0].NotePath, "stage-1 order must be unchanged")
}

func TestSearchRerankByDefaultAndOptOut(t *testing.T) {
	embSrv := embeddingServer(t, []float32{1, 0})

	t.Run("silence reranks", func(t *testing.T) {
		var ceCalls atomic.Int64
		env := searchRerankEnv(t, rerankFeatures(ceFlipsToB(t, &ceCalls), true), embSrv.URL)
		payload := rerankSearchCall(t, env)

		require.GreaterOrEqual(t, ceCalls.Load(), int64(1))
		require.Equal(t, "b.md", payload.Results[0].NotePath)
	})

	t.Run("explicit false wins over the default", func(t *testing.T) {
		var ceCalls atomic.Int64
		env := searchRerankEnv(t, rerankFeatures(ceFlipsToB(t, &ceCalls), true), embSrv.URL)
		payload := rerankSearchCallArgs(t, env, `{"query":"q","rerank":false}`)

		require.Equal(t, int64(0), ceCalls.Load(), "a caller must be able to decline the wait")
		require.Equal(t, "a.md", payload.Results[0].NotePath)
	})
}

func TestSearchWithoutRerankerIsUntouched(t *testing.T) {
	var ceCalls atomic.Int64
	rerankServer(t, &ceCalls, func(string) float64 { return 0.5 })
	embSrv := embeddingServer(t, []float32{1, 0})

	var feats features.Features
	feats.VectorSearch.Enabled = true // vector on, reranker off

	payload := rerankSearchCall(t, searchRerankEnv(t, feats, embSrv.URL))

	require.Equal(t, int64(0), ceCalls.Load(), "reranker must not be called when disabled")
	require.Len(t, payload.Results, 2)
	require.Equal(t, "a.md", payload.Results[0].NotePath, "stage-1 order must be unchanged")
}

// listSearchTools returns the advertised search / federated_search schemas.
func listSearchTools(t *testing.T, env *EnvMock) map[string]mcp.Tool {
	t.Helper()
	// searchRerankEnv is shaped for tools/call; listing also walks the note
	// corpus and the federation limits, so fill in what only that path touches.
	if env.FederationMaxDepthFunc == nil {
		env.FederationMaxDepthFunc = func() int { return 3 }
	}
	if env.FederatedGraphQLEnabledFunc == nil {
		env.FederatedGraphQLEnabledFunc = func() bool { return false }
	}
	if env.LatestNoteViewsFunc == nil {
		env.LatestNoteViewsFunc = appmodel.NewNoteViews
	}

	resp := callMCP(t, env, mcp.Request{JSONRPC: "2.0", Method: "tools/list", ID: 1})
	require.Nil(t, resp.Error)

	result, ok := resp.Result.(mcp.ListToolsResult)
	require.True(t, ok, "tools/list must return a ListToolsResult, got %T", resp.Result)

	out := map[string]mcp.Tool{}
	for _, tool := range result.Tools {
		if tool.Name == "search" || tool.Name == "federated_search" {
			out[tool.Name] = tool
		}
	}
	require.Len(t, out, 2, "both search tools must be advertised")
	return out
}

// An argument the instance cannot honour is worse than no argument: the agent
// spends a turn discovering it did nothing. So the schema only carries `rerank`
// where a cross-encoder is actually configured.
func TestToolsListHidesRerankArgWithoutSidecar(t *testing.T) {
	embSrv := embeddingServer(t, []float32{1, 0})

	var feats features.Features
	feats.VectorSearch.Enabled = true // vector on, no reranker configured

	for name, tool := range listSearchTools(t, searchRerankEnv(t, feats, embSrv.URL)) {
		require.NotContains(t, tool.InputSchema.Properties, "rerank",
			"%s must not advertise an argument this instance cannot honour", name)
	}
}

func TestToolsListAdvertisesRerankArgWithSidecar(t *testing.T) {
	embSrv := embeddingServer(t, []float32{1, 0})
	var ceCalls atomic.Int64
	ceURL := ceFlipsToB(t, &ceCalls)

	t.Run("opt-in instance states the cost and the default", func(t *testing.T) {
		tools := listSearchTools(t, searchRerankEnv(t, rerankFeatures(ceURL, false), embSrv.URL))
		for name, tool := range tools {
			prop, ok := tool.InputSchema.Properties["rerank"]
			require.True(t, ok, "%s must advertise rerank when a sidecar is configured", name)
			require.Equal(t, "boolean", prop.Type)
			require.Contains(t, prop.Description, "DEFAULT OFF",
				"the agent must be told what happens if it says nothing")
			require.Contains(t, prop.Description, "50",
				"the agent must be told the cost in the unit it controls — candidates")
		}
	})

	t.Run("default-on instance says so", func(t *testing.T) {
		tools := listSearchTools(t, searchRerankEnv(t, rerankFeatures(ceURL, true), embSrv.URL))
		require.Contains(t, tools["search"].InputSchema.Properties["rerank"].Description, "DEFAULT ON")
	})

	require.Equal(t, int64(0), ceCalls.Load(), "listing tools must never call the reranker")
}
