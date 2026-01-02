package generatenoteversionembedding_test

import (
	"context"
	"crypto/sha256"
	"testing"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	ctx := context.Background()

	t.Run("skips when vector search disabled", func(t *testing.T) {
		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: false},
				}
			},
			LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
		}

		err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err)
	})

	t.Run("skips when note not found in cache", func(t *testing.T) {
		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall},
				}
			},
			LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
			LatestNoteViewsFunc: func() *model.NoteViews {
				return &model.NoteViews{
					Map: map[string]*model.NoteView{}, // empty
				}
			},
		}

		err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 999})
		require.NoError(t, err)
	})

	t.Run("skips when embedding already up to date", func(t *testing.T) {
		noteView := &model.NoteView{
			VersionID: 1,
			Title:     "Test Note",
			Content:   []byte("Test content"),
			Permalink: "/test-note",
		}
		contentHash := sha256.Sum256([]byte(noteView.Title + string(noteView.Content)))

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall},
				}
			},
			LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
			LatestNoteViewsFunc: func() *model.NoteViews {
				return &model.NoteViews{
					Map: map[string]*model.NoteView{noteView.Permalink: noteView},
				}
			},
			GetNoteVersionEmbeddingFunc: func(ctx context.Context, versionID int64) (db.NoteVersionEmbedding, error) {
				return db.NoteVersionEmbedding{
					VersionID:   1,
					ContentHash: contentHash[:],
				}, nil
			},
		}

		err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err)
		require.Empty(t, env.UpsertNoteVersionEmbeddingCalls())
	})
}
