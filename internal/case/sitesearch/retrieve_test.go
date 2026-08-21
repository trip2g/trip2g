package sitesearch_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/sitesearch"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/openai"
)

// retrieveEnv builds an EnvMock with distinct latest and live corpora so tests
// can assert which corpus Retrieve consulted.
func retrieveEnv(t *testing.T, embURL string) *EnvMock {
	t.Helper()
	latest := chunkNote("latest.md", "/latest", "Latest")
	live := chunkNote("live.md", "/live", "Live")

	return &EnvMock{
		SearchLatestNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{NoteView: latest, URL: latest.Permalink}}, nil
		},
		SearchLiveNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{NoteView: live, URL: live.Permalink}}, nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{VectorSearch: features.VectorSearchConfig{Enabled: true}}
		},
		OpenAIFunc: func() *openai.Client { return openai.New("test-key", "test-model", embURL+"/v1") },
		LatestNoteChunksFunc: func() []appmodel.NoteChunk {
			return []appmodel.NoteChunk{
				{NotePath: "latest.md", ChunkIndex: 0, Content: "Latest\n\nlatest body", Embedding: []float32{1, 0}},
			}
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return []appmodel.NoteChunk{
				{NotePath: "live.md", ChunkIndex: 0, Content: "Live\n\nlive body", Embedding: []float32{1, 0}},
			}
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{PathMap: map[string]*appmodel.NoteView{"latest.md": latest}}
		},
		LiveNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{PathMap: map[string]*appmodel.NoteView{"live.md": live}}
		},
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
	}
}

func TestRetrieve_CorpusSelection(t *testing.T) {
	srv := newEmbeddingServer(t, []float32{1, 0})
	defer srv.Close()

	t.Run("live", func(t *testing.T) {
		env := retrieveEnv(t, srv.URL)
		results, merged, err := sitesearch.Retrieve(context.Background(), env, "q", false, nil)
		require.NoError(t, err)
		require.True(t, merged)
		require.Len(t, results, 1)
		require.Equal(t, "/live", results[0].URL)
		require.Empty(t, env.SearchLatestNotesCalls())
		require.Empty(t, env.LatestNoteChunksCalls())
	})

	t.Run("latest", func(t *testing.T) {
		env := retrieveEnv(t, srv.URL)
		results, merged, err := sitesearch.Retrieve(context.Background(), env, "q", true, nil)
		require.NoError(t, err)
		require.True(t, merged)
		require.Len(t, results, 1)
		require.Equal(t, "/latest", results[0].URL)
		require.Empty(t, env.SearchLiveNotesCalls())
		require.Empty(t, env.LiveNoteChunksCalls())
	})
}

// TestRetrieve_VectorResultsCarryChunkIndex asserts the shared core keeps the
// best-matching chunk index on results, both for vector-only hits and for
// hybrid hits that also matched the text lane. The MCP adapter builds chunk
// match_ids from it.
func TestRetrieve_VectorResultsCarryChunkIndex(t *testing.T) {
	srv := newEmbeddingServer(t, []float32{1, 0})
	defer srv.Close()

	noteA := chunkNote("a.md", "/a", "A")
	noteB := chunkNote("b.md", "/b", "B")

	env := retrieveEnv(t, srv.URL)
	env.SearchLiveNotesFunc = func(string) ([]appmodel.SearchResult, error) {
		// Text lane finds only A; B is vector-only.
		return []appmodel.SearchResult{{NoteView: noteA, URL: noteA.Permalink, HighlightedContent: []string{"text hit"}}}, nil
	}
	env.LiveNoteChunksFunc = func() []appmodel.NoteChunk {
		return []appmodel.NoteChunk{
			// For A the second chunk matches best; for B the first.
			{NotePath: "a.md", ChunkIndex: 0, Content: "A\n\nfar", Embedding: []float32{0, 1}},
			{NotePath: "a.md", ChunkIndex: 1, Content: "A\n\nnear", Embedding: []float32{1, 0}},
			{NotePath: "b.md", ChunkIndex: 0, Content: "B\n\nnear b", Embedding: []float32{0.9, 0.1}},
		}
	}
	env.LiveNoteViewsFunc = func() *appmodel.NoteViews {
		return &appmodel.NoteViews{PathMap: map[string]*appmodel.NoteView{"a.md": noteA, "b.md": noteB}}
	}

	results, _, err := sitesearch.Retrieve(context.Background(), env, "q", false, nil)
	require.NoError(t, err)

	byURL := map[string]appmodel.SearchResult{}
	for _, r := range results {
		byURL[r.URL] = r
	}

	require.Contains(t, byURL, "/a")
	require.NotNil(t, byURL["/a"].ChunkIndex, "hybrid hit must inherit the vector lane's chunk index")
	require.Equal(t, 1, *byURL["/a"].ChunkIndex)
	// Text-lane highlight must survive the merge.
	require.Equal(t, []string{"text hit"}, byURL["/a"].HighlightedContent)

	require.Contains(t, byURL, "/b")
	require.NotNil(t, byURL["/b"].ChunkIndex)
	require.Equal(t, 0, *byURL["/b"].ChunkIndex)
}

// TestRetrieve_DimensionMismatchSkipsStaleChunks: stale chunks with a different
// dimensionality than the query embedding must be skipped, not scored ~0.
func TestRetrieve_DimensionMismatchSkipsStaleChunks(t *testing.T) {
	srv := newEmbeddingServer(t, []float32{1, 0, 0})
	defer srv.Close()

	env := retrieveEnv(t, srv.URL)
	env.SearchLiveNotesFunc = func(string) ([]appmodel.SearchResult, error) { return nil, nil }
	// Stored chunks are 2-dim; the query is 3-dim.

	results, _, err := sitesearch.Retrieve(context.Background(), env, "q", false, nil)
	require.NoError(t, err)
	require.Empty(t, results, "dim-mismatched chunks must not produce candidates")
}
