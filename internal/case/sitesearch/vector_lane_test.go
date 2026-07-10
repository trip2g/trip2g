package sitesearch_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/sitesearch"
	"trip2g/internal/features"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/openai"
	"trip2g/internal/usertoken"
)

// newEmbeddingServer serves an OpenAI-compatible /embeddings endpoint that
// always returns the given query vector.
func newEmbeddingServer(t *testing.T, vector []float32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "index": 0, "embedding": vector},
			},
			"usage": map[string]any{"total_tokens": 3},
		})
	}))
}

func chunkNote(path, permalink, title string) *appmodel.NoteView {
	return &appmodel.NoteView{
		Path:      path,
		Permalink: permalink,
		Title:     title,
		Content:   []byte("content of " + title),
	}
}

// TestVectorSearch_DimensionMismatchYieldsNothing asserts that when the query
// embedding dimensionality differs from the stored chunk embeddings (a model
// switch without re-embedding), the vector lane must yield NO results (or an
// error) instead of an arbitrary top-K. Today dotSimilarity returns 0 for a
// length mismatch, so every chunk ties at 0 and the "top" vector candidates are
// just the alphabetically-first note paths — silent garbage.
func TestVectorSearch_DimensionMismatchYieldsNothing(t *testing.T) {
	// Query embedding is 3-dim; stored chunks are 4-dim.
	srv := newEmbeddingServer(t, []float32{1, 0, 0})
	defer srv.Close()

	notes := map[string]*appmodel.NoteView{
		"a.md": chunkNote("a.md", "/a", "A"),
		"b.md": chunkNote("b.md", "/b", "B"),
		"c.md": chunkNote("c.md", "/c", "C"),
	}
	chunks := []appmodel.NoteChunk{
		{NotePath: "a.md", Content: "A\n\nbody a", Embedding: []float32{1, 0, 0, 0}},
		{NotePath: "b.md", Content: "B\n\nbody b", Embedding: []float32{0, 1, 0, 0}},
		{NotePath: "c.md", Content: "C\n\nbody c", Embedding: []float32{0, 0, 1, 0}},
	}

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) { return &usertoken.Data{}, nil },
		SiteConfigFunc:       func(_ context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc:  func(_ string) ([]appmodel.SearchResult, error) { return nil, nil },
		FeaturesFunc: func() features.Features {
			return features.Features{VectorSearch: features.VectorSearchConfig{Enabled: true}}
		},
		OpenAIFunc:         func() *openai.Client { return openai.New("test-key", "test-model", srv.URL+"/v1") },
		LiveNoteChunksFunc: func() []appmodel.NoteChunk { return chunks },
		LiveNoteViewsFunc:  func() *appmodel.NoteViews { return &appmodel.NoteViews{PathMap: notes} },
		CanReadNoteFunc:    func(_ context.Context, _ *appmodel.NoteView) (bool, error) { return true, nil },
		LoggerFunc:         func() logger.Logger { return &logger.DummyLogger{} },
	}

	ctx := appreq.NewContext(context.Background(), &appreq.Request{})
	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "x"})
	require.NoError(t, err)
	require.Empty(t, conn.Nodes,
		"dimension mismatch must yield no vector candidates, not an alphabetical top-K")
}

// TestHybridSearch_ReadableBeyondCapSurfaces asserts that the fused-list size
// bound must not discard readable results before permission filtering. With >20
// fused results whose top 20 are unreadable, a readable result at rank 21+ must
// still surface. Today mergeResults hard-caps at 20 before CanReadNote runs, so
// the readable tail is gone before the filter ever sees it.
func TestHybridSearch_ReadableBeyondCapSurfaces(t *testing.T) {
	embSrv := newEmbeddingServer(t, []float32{1, 0})
	defer embSrv.Close()

	// 24 text results; a single vector chunk for the first one triggers the
	// hybrid merge path (and its cap). Only the last-ranked note is readable.
	var textResults []appmodel.SearchResult
	for i := 1; i <= 24; i++ {
		path := fmt.Sprintf("n%02d.md", i)
		permalink := fmt.Sprintf("/n%02d", i)
		title := fmt.Sprintf("N%02d", i)
		textResults = append(textResults, appmodel.SearchResult{
			URL:      permalink,
			NoteView: chunkNote(path, permalink, title),
		})
	}
	readable := textResults[23]

	firstNote := textResults[0].NoteView
	chunks := []appmodel.NoteChunk{
		{NotePath: firstNote.Path, Content: firstNote.Title + "\n\nbody", Embedding: []float32{1, 0}},
	}

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) { return &usertoken.Data{}, nil },
		SiteConfigFunc:       func(_ context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc:  func(_ string) ([]appmodel.SearchResult, error) { return textResults, nil },
		FeaturesFunc: func() features.Features {
			return features.Features{VectorSearch: features.VectorSearchConfig{Enabled: true}}
		},
		OpenAIFunc:         func() *openai.Client { return openai.New("test-key", "test-model", embSrv.URL+"/v1") },
		LiveNoteChunksFunc: func() []appmodel.NoteChunk { return chunks },
		LiveNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{PathMap: map[string]*appmodel.NoteView{firstNote.Path: firstNote}}
		},
		CanReadNoteFunc: func(_ context.Context, nv *appmodel.NoteView) (bool, error) {
			return nv.Path == readable.NoteView.Path, nil
		},
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
	}

	ctx := appreq.NewContext(context.Background(), &appreq.Request{})
	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "x"})
	require.NoError(t, err)

	var readableURLs []string
	for _, n := range conn.Nodes {
		if n.NoteView != nil {
			readableURLs = append(readableURLs, n.URL)
		}
	}
	require.Contains(t, readableURLs, readable.URL,
		"readable result ranked beyond the candidate cap must surface when higher-ranked results are unreadable")
}

// TestRerank_OutputKTruncatesAfterPermissionFilter asserts that OutputK must not
// discard readable results before permission filtering. With output_k=2 and the
// top blended result unreadable, the readable third result must still surface.
// Today rerankResults truncates to OutputK first, so the third result is gone
// before CanReadNote ever runs.
func TestRerank_OutputKTruncatesAfterPermissionFilter(t *testing.T) {
	embSrv := newEmbeddingServer(t, []float32{1, 0})
	defer embSrv.Close()

	// Fake TEI rerank endpoint (bare-array contract the client currently parses):
	// CE scores preserve the vector-lane order a > b > c.
	rerankSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"index":0,"score":0.9},{"index":1,"score":0.5},{"index":2,"score":0.1}]`))
	}))
	defer rerankSrv.Close()

	notes := map[string]*appmodel.NoteView{
		"a.md": chunkNote("a.md", "/a", "A"),
		"b.md": chunkNote("b.md", "/b", "B"),
		"c.md": chunkNote("c.md", "/c", "C"),
	}
	// Similarities against the [1,0] query: a=1.0, b=0.8, c=0.6.
	chunks := []appmodel.NoteChunk{
		{NotePath: "a.md", Content: "A\n\nbody a", Embedding: []float32{1, 0}},
		{NotePath: "b.md", Content: "B\n\nbody b", Embedding: []float32{0.8, 0.6}},
		{NotePath: "c.md", Content: "C\n\nbody c", Embedding: []float32{0.6, 0.8}},
	}

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) { return &usertoken.Data{}, nil },
		SiteConfigFunc:       func(_ context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc:  func(_ string) ([]appmodel.SearchResult, error) { return nil, nil },
		FeaturesFunc: func() features.Features {
			return features.Features{VectorSearch: features.VectorSearchConfig{
				Enabled: true,
				Reranker: features.RerankerConfig{
					Enabled:     true,
					BaseURL:     rerankSrv.URL + "/rerank",
					TopN:        10,
					OutputK:     2,
					BlendWeight: 0.5,
				},
			}}
		},
		OpenAIFunc:         func() *openai.Client { return openai.New("test-key", "test-model", embSrv.URL+"/v1") },
		LiveNoteChunksFunc: func() []appmodel.NoteChunk { return chunks },
		LiveNoteViewsFunc:  func() *appmodel.NoteViews { return &appmodel.NoteViews{PathMap: notes} },
		CanReadNoteFunc: func(_ context.Context, nv *appmodel.NoteView) (bool, error) {
			return nv.Path != "a.md", nil // top blended result is unreadable
		},
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
	}

	ctx := appreq.NewContext(context.Background(), &appreq.Request{})
	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "x"})
	require.NoError(t, err)

	var readableURLs []string
	for _, n := range conn.Nodes {
		if n.NoteView != nil {
			readableURLs = append(readableURLs, n.URL)
		}
	}
	require.Contains(t, readableURLs, "/c",
		"readable /c must surface when the top result is hidden; OutputK must not truncate before permission filtering")
}
