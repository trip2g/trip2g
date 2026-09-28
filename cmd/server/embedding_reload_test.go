package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"trip2g/internal/appconfig"
	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/configregistry"
	"trip2g/internal/features"
	"trip2g/internal/noteloader"
	"trip2g/internal/openai"
	"trip2g/internal/throttle"

	"github.com/stretchr/testify/require"
)

// newEmbeddingServer answers every OpenAI-compatible /embeddings request with
// one fixed vector per input, so the job can run against a real database
// without a real embedding model.
func newEmbeddingServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input json.RawMessage `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var inputs []string
		if err := json.Unmarshal(req.Input, &inputs); err != nil {
			var single string
			_ = json.Unmarshal(req.Input, &single)
			inputs = []string{single}
		}
		data := make([]map[string]any, len(inputs))
		for i := range inputs {
			data[i] = map[string]any{"object": "embedding", "index": i, "embedding": []float32{0.6, 0.8, 0}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data":   data,
			"usage":  map[string]any{"total_tokens": len(inputs)},
		})
	}))
}

// newEmbeddingTestApp wires a real-database app with the production note
// loader envs and the embedding job's dependencies.
func newEmbeddingTestApp(t *testing.T, embeddingURL string) *app {
	t.Helper()

	a := newTxTestApp(t)
	a.config = &appconfig.Config{
		PublicURL:               "https://example.com",
		EmbeddingReloadInterval: time.Millisecond,
		Features: features.Features{
			VectorSearch: features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall},
		},
	}
	a.openaiClient = openai.New("test-key", "test-model", embeddingURL+"/v1")
	a.SiteConfigBuilder = configregistry.NewSiteConfigBuilder(a)
	a.latestNoteLoader = noteloader.New("latest", makeLatestNoteLoaderWrapper(a), a.config.MDLoaderConfig)
	a.liveNoteLoader = noteloader.New("live", makeLiveNoteLoaderWrapper(a), a.config.MDLoaderConfig)
	a.embeddingReload = throttle.New(a.config.EmbeddingReloadInterval, a.reloadNoteEmbeddings)
	return a
}

// TestEmbeddingJob_ReachesInMemoryVectors pins the fix for vector search
// returning nothing after a push: notes are loaded first, the embedding job
// writes the whole-note embedding and the chunks afterwards, and search and
// similar-notes read the loader's in-memory copies — so the job's signal must
// bring both into memory without waiting for the next note reload.
func TestEmbeddingJob_ReachesInMemoryVectors(t *testing.T) {
	srv := newEmbeddingServer(t)
	defer srv.Close()

	a := newEmbeddingTestApp(t, srv.URL)
	ctx := context.Background()

	body := strings.Repeat("a sentence about gardening tools and their upkeep ", 30)
	versionID := insertLatestNote(t, a, "guide.md", "---\ntitle: Guide\n---\n"+body)

	require.NoError(t, a.latestNoteLoader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))
	require.Empty(t, a.LatestNoteChunks(), "no chunks exist before the job runs")
	require.Empty(t, a.LatestNoteViews().GetByVersionID(versionID).Embedding)

	err := generatenoteversionembedding.Resolve(ctx, a, generatenoteversionembedding.Params{VersionID: versionID})
	require.NoError(t, err)

	stored, err := a.Queries.GetAllLatestNoteChunksWithEmbeddings(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, stored, "the job persisted chunk embeddings")

	require.Eventually(t, func() bool {
		return len(a.LatestNoteChunks()) == len(stored)
	}, 5*time.Second, 5*time.Millisecond, "chunks saved by the embedding job must reach vector search without a note reload")
	for _, c := range a.LatestNoteChunks() {
		require.Equal(t, versionID, c.VersionID)
		require.Equal(t, "guide.md", c.NotePath)
		require.Equal(t, []float32{0.6, 0.8, 0}, c.Embedding)
	}

	note := a.LatestNoteViews().GetByVersionID(versionID)
	require.NotNil(t, note)
	require.Equal(t, []float32{0.6, 0.8, 0}, note.Embedding, "the whole-note embedding must reach similar-notes without a note reload")
}
