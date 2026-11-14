package regeneratenoteembeddings_test

import (
	"context"
	"crypto/sha256"
	"strconv"
	"testing"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/case/cronjob/regeneratenoteembeddings"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/mdchunk"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	ctx := context.Background()

	noteView := &model.NoteView{
		VersionID: 1,
		Title:     "Test Note",
		Content:   []byte("---\ntags: [x]\n---\nTest content"),
		Permalink: "/test-note",
	}

	// freshChunks builds the chunk rows the generator stores for noteView.
	freshChunks := func(cfg features.VectorSearchConfig) []db.GetNoteVersionChunkHashesByVersionIDsRow {
		split := mdchunk.Split(noteView.Title, noteView.Content)
		rows := make([]db.GetNoteVersionChunkHashesByVersionIDsRow, len(split))
		for i, c := range split {
			h := sha256.Sum256([]byte(c.Content + modelFingerprint(cfg)))
			rows[i] = db.GetNoteVersionChunkHashesByVersionIDsRow{VersionID: 1, ChunkIndex: int64(c.Index), ContentHash: h[:]}
		}
		return rows
	}

	newEnv := func(cfg features.VectorSearchConfig, storedHash []byte) *EnvMock {
		chunks := freshChunks(cfg)
		return &EnvMock{
			GetNoteVersionChunkHashesByVersionIDsFunc: func(ctx context.Context, versionIDs []int64) ([]db.GetNoteVersionChunkHashesByVersionIDsRow, error) {
				return chunks, nil
			},
			FeaturesFunc: func() features.Features {
				return features.Features{VectorSearch: cfg}
			},
			LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
			LatestNoteViewsFunc: func() *model.NoteViews {
				return &model.NoteViews{List: []*model.NoteView{noteView}}
			},
			GetNoteVersionEmbeddingsByVersionIDsFunc: func(ctx context.Context, versionIDs []int64) ([]db.NoteVersionEmbedding, error) {
				return []db.NoteVersionEmbedding{{VersionID: 1, ContentHash: storedHash}}, nil
			},
			EnqueueGenerateNoteVersionEmbeddingFunc: func(ctx context.Context, versionID int64) error {
				return nil
			},
		}
	}

	t.Run("counts fingerprinted embedding as up to date", func(t *testing.T) {
		// The generator stores sha256(title + stripped content + model fingerprint).
		// The sweep must compute the identical hash, otherwise every fresh row is
		// re-enqueued on each run.
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}
		storedHash := generatenoteversionembedding.NoteContentHash(noteView.Title, noteView.Content, cfg)

		env := newEnv(cfg, storedHash)
		result, err := regeneratenoteembeddings.Resolve(ctx, env)
		require.NoError(t, err)
		require.Equal(t, 1, result.UpToDateCount)
		require.Equal(t, 0, result.EnqueuedCount)
		require.Empty(t, env.EnqueueGenerateNoteVersionEmbeddingCalls())
	})

	t.Run("enqueues a note whose whole-note row is current but chunks are missing", func(t *testing.T) {
		// A job that saved the whole-note row and then failed on the chunk
		// embeddings leaves the note invisible to chunk search. Its content never
		// changes again, so only the sweep can bring it back.
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}
		storedHash := generatenoteversionembedding.NoteContentHash(noteView.Title, noteView.Content, cfg)

		env := newEnv(cfg, storedHash)
		env.GetNoteVersionChunkHashesByVersionIDsFunc = func(ctx context.Context, versionIDs []int64) ([]db.GetNoteVersionChunkHashesByVersionIDsRow, error) {
			return nil, nil
		}
		result, err := regeneratenoteembeddings.Resolve(ctx, env)
		require.NoError(t, err)
		require.Equal(t, 0, result.UpToDateCount)
		require.Equal(t, 1, result.EnqueuedCount)
		require.Len(t, env.EnqueueGenerateNoteVersionEmbeddingCalls(), 1)
	})

	t.Run("enqueues a note that has no embedding at all", func(t *testing.T) {
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}

		env := newEnv(cfg, nil)
		env.GetNoteVersionEmbeddingsByVersionIDsFunc = func(ctx context.Context, versionIDs []int64) ([]db.NoteVersionEmbedding, error) {
			return nil, nil
		}
		result, err := regeneratenoteembeddings.Resolve(ctx, env)
		require.NoError(t, err)
		require.Equal(t, 1, result.EnqueuedCount)
		require.Equal(t, int64(1), env.EnqueueGenerateNoteVersionEmbeddingCalls()[0].VersionID)
	})

	t.Run("enqueues a note whose whole-note row has another model", func(t *testing.T) {
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}
		storedHash := generatenoteversionembedding.NoteContentHash(noteView.Title, noteView.Content, cfg)

		env := newEnv(cfg, storedHash)
		env.GetNoteVersionEmbeddingsByVersionIDsFunc = func(ctx context.Context, versionIDs []int64) ([]db.NoteVersionEmbedding, error) {
			return []db.NoteVersionEmbedding{{VersionID: 1, ContentHash: storedHash, ModelID: int64(features.EmbeddingModelSmall) + 1}}, nil
		}
		result, err := regeneratenoteembeddings.Resolve(ctx, env)
		require.NoError(t, err)
		require.Equal(t, 1, result.EnqueuedCount)
	})

	t.Run("force enqueues a note the hash calls up to date", func(t *testing.T) {
		// The content hash covers title, content and the model fingerprint — not
		// how the note is split into chunks. After a chunker change every note
		// looks up to date, so the repair lever has to ignore the hash.
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}
		storedHash := generatenoteversionembedding.NoteContentHash(noteView.Title, noteView.Content, cfg)

		env := newEnv(cfg, storedHash)
		result, err := regeneratenoteembeddings.ResolveForced(ctx, env)
		require.NoError(t, err)
		require.Equal(t, 1, result.EnqueuedCount)
		require.Equal(t, 0, result.UpToDateCount)
		require.Len(t, env.EnqueueGenerateNoteVersionEmbeddingCalls(), 1)
	})

	t.Run("enqueues when embedding params change", func(t *testing.T) {
		// The stored row was hashed under the old dimensions; after a config change
		// the fingerprint differs, so the sweep must re-enqueue the note.
		oldCfg := features.VectorSearchConfig{
			Enabled: true, ModelName: "my-model", Model: features.EmbeddingModelCustom,
			Dimensions: 512, MaxTokens: 512,
		}
		newCfg := oldCfg
		newCfg.Dimensions = 1024
		storedHash := generatenoteversionembedding.NoteContentHash(noteView.Title, noteView.Content, oldCfg)

		env := newEnv(newCfg, storedHash)
		result, err := regeneratenoteembeddings.Resolve(ctx, env)
		require.NoError(t, err)
		require.Equal(t, 0, result.UpToDateCount)
		require.Equal(t, 1, result.EnqueuedCount)
		require.Len(t, env.EnqueueGenerateNoteVersionEmbeddingCalls(), 1)
	})
}

func modelFingerprint(cfg features.VectorSearchConfig) string {
	return "\x00model=" + cfg.ResolvedModelName() +
		"\x00passage_prefix=" + cfg.ResolvedPassagePrefix() +
		"\x00dimensions=" + strconv.Itoa(cfg.ResolvedDimensions()) +
		"\x00max_input_tokens=" + strconv.Itoa(cfg.ResolvedMaxInputTokens())
}
