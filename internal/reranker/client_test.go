package reranker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRerankReordersByScore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rerank", r.URL.Path)
		// Server returns results best-first; doc index 2 is most relevant.
		_, _ = w.Write([]byte(`{"results":[{"index":2,"relevance_score":0.9},{"index":0,"relevance_score":0.5},{"index":1,"relevance_score":0.1}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/rerank", "BAAI/bge-reranker-v2-m3")
	order, err := c.Rerank(context.Background(), "q", []string{"doc0", "doc1", "doc2"})
	require.NoError(t, err)
	require.Equal(t, []int{2, 0, 1}, order)
}

func TestRerankEmptyDocs(t *testing.T) {
	c := New("http://unused/rerank", "m")
	order, err := c.Rerank(context.Background(), "q", nil)
	require.NoError(t, err)
	require.Empty(t, order)
}

func TestRerankDropsOutOfRangeIndices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":5,"relevance_score":0.9},{"index":0,"relevance_score":0.5}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/rerank", "m")
	order, err := c.Rerank(context.Background(), "q", []string{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, []int{0}, order)
}
