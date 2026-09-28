package noteloader_test

import (
	"context"
	"testing"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"

	"github.com/stretchr/testify/require"
)

// TestLoader_ReloadEmbeddings pins that embeddings and chunks written after a
// Load reach the published snapshot through ReloadEmbeddings alone: no note is
// re-rendered, the NoteView pointers stay the same, and the generation counter
// does not move.
func TestLoader_ReloadEmbeddings(t *testing.T) {
	notes := []noteloader.RawNote{
		{Path: "a.md", PathID: 1, VersionID: 11, Content: "---\ntitle: A\n---\nbody a"},
		{Path: "b.md", PathID: 2, VersionID: 12, Content: "---\ntitle: B\n---\nbody b"},
	}

	var embeddings []noteloader.RawNoteEmbedding
	var chunks []noteloader.RawNoteChunk
	env := makeMinimalEnv(notes, func() bool { return false })
	env.RawNoteEmbeddingsFunc = func(_ context.Context) ([]noteloader.RawNoteEmbedding, error) {
		return embeddings, nil
	}
	env.RawNoteChunksFunc = func(_ context.Context) ([]noteloader.RawNoteChunk, error) {
		return chunks, nil
	}

	loader := noteloader.New("test", env, mdloader.Config{})
	ctx := context.Background()
	require.NoError(t, loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))

	before := loader.NoteViews()
	require.Empty(t, before.GetByVersionID(11).Embedding)
	require.Empty(t, loader.NoteChunks())
	generation := loader.Generation()

	vector := []float32{0.6, 0.8, 0}
	embeddings = []noteloader.RawNoteEmbedding{{VersionID: 11, Embedding: model.Float32SliceToBytes(vector)}}
	chunks = []noteloader.RawNoteChunk{
		{VersionID: 11, ChunkIndex: 0, Content: "A\n\nbody a", Embedding: model.Float32SliceToBytes(vector), Path: "a.md"},
		{VersionID: 12, ChunkIndex: 0, Content: "B\n\nbody b", Embedding: nil, Path: "b.md"},
	}

	require.NoError(t, loader.ReloadEmbeddings(ctx))

	after := loader.NoteViews()
	require.Same(t, before.GetByVersionID(11), after.GetByVersionID(11), "no note is re-rendered")
	require.Equal(t, vector, after.GetByVersionID(11).Embedding)
	require.Empty(t, after.GetByVersionID(12).Embedding)
	require.Equal(t, generation, loader.Generation())

	loaded := loader.NoteChunks()
	require.Len(t, loaded, 1, "a chunk row without an embedding is not searchable")
	require.Equal(t, int64(11), loaded[0].VersionID)
	require.Equal(t, "a.md", loaded[0].NotePath)
	require.Equal(t, vector, loaded[0].Embedding)
	require.Len(t, env.RawNotesCalls(), 1, "ReloadEmbeddings must not re-read note content")
}
