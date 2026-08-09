package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/mcp"
	"trip2g/internal/features"
	"trip2g/internal/logger"
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
	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"q"}`),
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)
	resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
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

func TestSearchAppliesRerankerWhenConfigured(t *testing.T) {
	// Both lanes rank A first; the CE strongly prefers B and a high blend
	// weight lets it flip the order — MCP search must surface B on top.
	var ceCalls atomic.Int64
	ceSrv := rerankServer(t, &ceCalls, func(doc string) float64 {
		if strings.Contains(doc, "beta") {
			return 0.99
		}
		return 0.01
	})
	embSrv := embeddingServer(t, []float32{1, 0})

	var feats features.Features
	feats.VectorSearch.Enabled = true
	feats.VectorSearch.Reranker = features.RerankerConfig{
		Enabled:        true,
		BaseURL:        ceSrv.URL,
		Model:          "test",
		TopN:           50,
		OutputK:        20,
		BlendWeight:    0.9,
		TimeoutSeconds: 5,
	}

	payload := rerankSearchCall(t, searchRerankEnv(t, feats, embSrv.URL))

	require.GreaterOrEqual(t, ceCalls.Load(), int64(1), "reranker must be called")
	require.Len(t, payload.Results, 2)
	require.Equal(t, "b.md", payload.Results[0].NotePath, "CE-preferred note must rank first")
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
