package reranker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRerankReturnsScores(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rerank", r.URL.Path)
		// TEI returns a bare array with "score" (not a wrapped {"results":[...]} with "relevance_score").
		_, _ = w.Write([]byte(`[{"index":2,"score":0.9},{"index":0,"score":0.5},{"index":1,"score":0.1}]`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/rerank", "BAAI/bge-reranker-v2-m3")
	got, err := c.Rerank(context.Background(), "q", []string{"doc0", "doc1", "doc2"})
	require.NoError(t, err)
	require.Equal(t, []Result{{Index: 2, Score: 0.9}, {Index: 0, Score: 0.5}, {Index: 1, Score: 0.1}}, got)
}

func TestRerankEmptyDocs(t *testing.T) {
	c := New("http://unused/rerank", "m")
	got, err := c.Rerank(context.Background(), "q", nil)
	require.NoError(t, err)
	require.Empty(t, got)
}

// TestRerankAgainstBundledSidecar asserts the client speaks the wire contract of
// the bundled sidecar. The handler faithfully emulates reranker-server/server.py:
// FastAPI rejects a request without the required "texts" field with 422, and the
// response is a bare [{"index","score"}] array sorted by score descending.
func TestRerankAgainstBundledSidecar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string   `json:"query"`
			Texts []string `json:"texts"`
			Model string   `json:"model"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		if req.Texts == nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"detail":[{"type":"missing","loc":["body","texts"],"msg":"Field required"}]}`))
			return
		}
		type item struct {
			Index int     `json:"index"`
			Score float64 `json:"score"`
		}
		results := make([]item, len(req.Texts))
		for i := range req.Texts {
			results[i] = item{Index: i, Score: 1.0 / float64(i+1)}
		}
		// Already in descending score order, matching server.py's sort.
		_ = json.NewEncoder(w).Encode(results)
	}))
	defer srv.Close()

	c := New(srv.URL+"/rerank", "BAAI/bge-reranker-v2-m3")
	got, err := c.Rerank(context.Background(), "q", []string{"doc0", "doc1"})
	require.NoError(t, err)
	require.Equal(t, []Result{{Index: 0, Score: 1.0}, {Index: 1, Score: 0.5}}, got)
}

func TestRerankDropsOutOfRangeIndices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"index":5,"score":0.9},{"index":0,"score":0.5}]`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/rerank", "m")
	got, err := c.Rerank(context.Background(), "q", []string{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, []Result{{Index: 0, Score: 0.5}}, got)
}
