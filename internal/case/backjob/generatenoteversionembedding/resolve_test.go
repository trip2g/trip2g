package generatenoteversionembedding_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/openai"

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
			GetNoteVersionChunksFunc: func(ctx context.Context, versionID int64) ([]db.NoteVersionChunk, error) {
				return []db.NoteVersionChunk{{VersionID: 1}}, nil
			},
		}

		err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err)
		require.Empty(t, env.UpsertNoteVersionEmbeddingCalls())
	})

	t.Run("re-embeds when configured model differs from stored row", func(t *testing.T) {
		// Content is unchanged (hash matches), but the row was embedded with a
		// different model. Vectors from different models are incomparable, so the
		// embedding must be regenerated — the content-hash skip must not apply.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"data": []map[string]any{
					{"object": "embedding", "index": 0, "embedding": []float32{0.1, 0.2, 0.3}},
				},
				"usage": map[string]any{"total_tokens": 3},
			})
		}))
		defer srv.Close()

		noteView := &model.NoteView{
			VersionID:     1,
			Title:         "Test Note",
			Content:       []byte("Test content"),
			Permalink:     "/test-note",
			ExcludeSearch: true, // keep the test on the whole-note path, no chunk phase
		}
		contentHash := sha256.Sum256([]byte(noteView.Title + string(noteView.Content)))

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelBGEM3},
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
					ModelID:     int64(features.EmbeddingModelSmall), // embedded by a different model
				}, nil
			},
			OpenAIFunc: func() *openai.Client {
				return openai.New("test-key", "bge-m3", srv.URL+"/v1")
			},
			UpsertNoteVersionEmbeddingFunc: func(ctx context.Context, arg db.UpsertNoteVersionEmbeddingParams) error {
				return nil
			},
		}

		err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err)
		require.Len(t, env.UpsertNoteVersionEmbeddingCalls(), 1,
			"model switch must regenerate the embedding even when the content hash matches")
	})
}
