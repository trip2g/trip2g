package regeneratenoteembeddings_test

import (
	"context"
	"testing"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/case/cronjob/regeneratenoteembeddings"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
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

	newEnv := func(cfg features.VectorSearchConfig, storedHash []byte) *EnvMock {
		return &EnvMock{
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
