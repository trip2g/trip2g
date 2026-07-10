package generatenoteversionembedding_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/openai"

	"github.com/stretchr/testify/require"
)

// modelFingerprint mirrors the production fingerprint mixed into content
// hashes, so tests can build an "up to date" stored hash for a given config.
func modelFingerprint(cfg features.VectorSearchConfig) string {
	return "\x00model=" + cfg.ResolvedModelName() +
		"\x00passage_prefix=" + cfg.ResolvedPassagePrefix() +
		"\x00dimensions=" + strconv.Itoa(cfg.ResolvedDimensions()) +
		"\x00max_input_tokens=" + strconv.Itoa(cfg.ResolvedMaxInputTokens())
}

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
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}
		contentHash := sha256.Sum256([]byte(noteView.Title + string(noteView.Content) + modelFingerprint(cfg)))

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{VectorSearch: cfg}
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
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelBGEM3}
		// Hash matches the configured model's fingerprint: the stored ModelID is
		// the only difference, so the test isolates the model-mismatch check.
		contentHash := sha256.Sum256([]byte(noteView.Title + string(noteView.Content) + modelFingerprint(cfg)))

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{VectorSearch: cfg}
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

	t.Run("re-embeds when embedding params change for same custom model", func(t *testing.T) {
		// Same custom model name, unchanged content, but dimensions or
		// max_input_tokens changed. Both alter what gets stored (vector width;
		// truncation of long inputs), so a row hashed without them must be
		// regenerated — the content-hash skip must not apply.
		tests := []struct {
			name string
			cfg  features.VectorSearchConfig
		}{
			{
				name: "dimensions changed",
				cfg: features.VectorSearchConfig{
					Enabled: true, ModelName: "my-model", Model: features.EmbeddingModelCustom,
					Dimensions: 1024, MaxTokens: 512,
				},
			},
			{
				name: "max_input_tokens changed",
				cfg: features.VectorSearchConfig{
					Enabled: true, ModelName: "my-model", Model: features.EmbeddingModelCustom,
					Dimensions: 512, MaxTokens: 8192,
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
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
				// Hash as written by a fingerprint that covered only model name and
				// passage prefix — i.e. before dimensions/max_input_tokens were mixed in.
				staleHash := sha256.Sum256([]byte(noteView.Title + string(noteView.Content) +
					"\x00model=my-model\x00passage_prefix="))

				env := &EnvMock{
					FeaturesFunc: func() features.Features {
						return features.Features{VectorSearch: tc.cfg}
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
							ContentHash: staleHash[:],
							ModelID:     0, // custom models all store ModelID 0
						}, nil
					},
					OpenAIFunc: func() *openai.Client {
						return openai.New("test-key", "my-model", srv.URL+"/v1")
					},
					UpsertNoteVersionEmbeddingFunc: func(ctx context.Context, arg db.UpsertNoteVersionEmbeddingParams) error {
						return nil
					},
				}

				err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
				require.NoError(t, err)
				require.Len(t, env.UpsertNoteVersionEmbeddingCalls(), 1,
					"changed embedding params must regenerate the embedding even when content is unchanged")
			})
		}
	})
}
