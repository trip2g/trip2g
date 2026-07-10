package generatenoteversionembedding_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/mdchunk"
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

		// The stored chunk set must be complete and fresh: chunk verification
		// compares per-chunk hashes, not mere presence.
		chunks := mdchunk.Split(noteView.Title, noteView.Content)
		existingChunks := make([]db.NoteVersionChunk, len(chunks))
		for i, c := range chunks {
			h := sha256.Sum256([]byte(c.Content + modelFingerprint(cfg)))
			existingChunks[i] = db.NoteVersionChunk{VersionID: 1, ChunkIndex: int64(c.Index), ContentHash: h[:]}
		}

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
				return existingChunks, nil
			},
			DeleteNoteVersionChunksBeyondFunc: func(ctx context.Context, arg db.DeleteNoteVersionChunksBeyondParams) error {
				return nil
			},
		}

		err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err)
		require.Empty(t, env.UpsertNoteVersionEmbeddingCalls())
		require.Empty(t, env.UpsertNoteVersionChunkCalls())
	})

	t.Run("re-embeds stale or missing chunks when whole-note hash matches", func(t *testing.T) {
		// The whole-note row is saved before chunk upserts, so a retry after a
		// partial failure sees a matching whole-note hash plus an incomplete or
		// stale chunk set. The skip must verify the chunks, not just their presence.
		cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}

		// Two paragraphs big enough to split into at least two chunks.
		para := strings.Repeat("some fairly long sentence about nothing in particular ", 40)
		content := []byte(para + "\n\n" + para)
		noteView := &model.NoteView{
			VersionID: 1,
			Title:     "Test Note",
			Content:   content,
			Permalink: "/test-note",
		}
		chunks := mdchunk.Split(noteView.Title, content)
		require.GreaterOrEqual(t, len(chunks), 2, "test setup: note must split into multiple chunks")
		freshChunkHash := func(c mdchunk.Chunk) []byte {
			h := sha256.Sum256([]byte(c.Content + modelFingerprint(cfg)))
			return h[:]
		}

		tests := []struct {
			name     string
			existing []db.NoteVersionChunk
		}{
			{
				name: "stale chunk hash",
				existing: []db.NoteVersionChunk{
					{VersionID: 1, ChunkIndex: 0, ContentHash: []byte("stale")},
					{VersionID: 1, ChunkIndex: 1, ContentHash: freshChunkHash(chunks[1])},
				},
			},
			{
				name: "incomplete chunk set",
				existing: []db.NoteVersionChunk{
					{VersionID: 1, ChunkIndex: 0, ContentHash: freshChunkHash(chunks[0])},
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req struct {
						Input []string `json:"input"`
					}
					_ = json.NewDecoder(r.Body).Decode(&req)
					data := make([]map[string]any, len(req.Input))
					for i := range req.Input {
						data[i] = map[string]any{"object": "embedding", "index": i, "embedding": []float32{0.1, 0.2, 0.3}}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{
						"object": "list",
						"data":   data,
						"usage":  map[string]any{"total_tokens": len(req.Input)},
					})
				}))
				defer srv.Close()

				env := &EnvMock{
					FeaturesFunc: func() features.Features { return features.Features{VectorSearch: cfg} },
					LoggerFunc:   func() logger.Logger { return &logger.TestLogger{} },
					LatestNoteViewsFunc: func() *model.NoteViews {
						return &model.NoteViews{Map: map[string]*model.NoteView{noteView.Permalink: noteView}}
					},
					GetNoteVersionEmbeddingFunc: func(ctx context.Context, versionID int64) (db.NoteVersionEmbedding, error) {
						return db.NoteVersionEmbedding{
							VersionID:   1,
							ContentHash: generatenoteversionembedding.NoteContentHash(noteView.Title, content, cfg),
						}, nil
					},
					GetNoteVersionChunksFunc: func(ctx context.Context, versionID int64) ([]db.NoteVersionChunk, error) {
						return tc.existing, nil
					},
					OpenAIFunc: func() *openai.Client {
						return openai.New("test-key", "test-model", srv.URL+"/v1")
					},
					UpsertNoteVersionChunkFunc: func(ctx context.Context, arg db.UpsertNoteVersionChunkParams) error {
						return nil
					},
					DeleteNoteVersionChunksBeyondFunc: func(ctx context.Context, arg db.DeleteNoteVersionChunksBeyondParams) error {
						return nil
					},
				}

				err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
				require.NoError(t, err)
				require.Empty(t, env.UpsertNoteVersionEmbeddingCalls(),
					"whole-note embedding is up to date and must not be regenerated")
				require.NotEmpty(t, env.UpsertNoteVersionChunkCalls(),
					"stale or missing chunks must be re-embedded even when the whole-note hash matches")
			})
		}
	})

	t.Run("truncates oversized chunks to the token budget", func(t *testing.T) {
		// A custom model with a small max_input_tokens must not send chunks past
		// the budget: the server rejects them with HTTP 400 and the job retries
		// forever. Chunk text must be truncated like the whole-note input is.
		var requests [][]string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Input []string `json:"input"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			requests = append(requests, req.Input)
			data := make([]map[string]any, len(req.Input))
			for i := range req.Input {
				data[i] = map[string]any{"object": "embedding", "index": i, "embedding": []float32{0.1, 0.2, 0.3}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"data":   data,
				"usage":  map[string]any{"total_tokens": len(req.Input)},
			})
		}))
		defer srv.Close()

		noteView := &model.NoteView{
			VersionID: 1,
			Title:     "Test Note",
			Content:   []byte(strings.Repeat("some fairly long sentence about nothing in particular ", 20)),
			Permalink: "/test-note",
		}
		cfg := features.VectorSearchConfig{
			Enabled: true, ModelName: "my-model", Model: features.EmbeddingModelCustom,
			Dimensions: 3, MaxTokens: 40,
		}

		env := &EnvMock{
			FeaturesFunc: func() features.Features { return features.Features{VectorSearch: cfg} },
			LoggerFunc:   func() logger.Logger { return &logger.TestLogger{} },
			LatestNoteViewsFunc: func() *model.NoteViews {
				return &model.NoteViews{Map: map[string]*model.NoteView{noteView.Permalink: noteView}}
			},
			GetNoteVersionEmbeddingFunc: func(ctx context.Context, versionID int64) (db.NoteVersionEmbedding, error) {
				return db.NoteVersionEmbedding{}, sql.ErrNoRows
			},
			GetNoteVersionChunksFunc: func(ctx context.Context, versionID int64) ([]db.NoteVersionChunk, error) {
				return nil, nil
			},
			OpenAIFunc: func() *openai.Client {
				return openai.New("test-key", "my-model", srv.URL+"/v1")
			},
			UpsertNoteVersionEmbeddingFunc: func(ctx context.Context, arg db.UpsertNoteVersionEmbeddingParams) error {
				return nil
			},
			UpsertNoteVersionChunkFunc: func(ctx context.Context, arg db.UpsertNoteVersionChunkParams) error {
				return nil
			},
			DeleteNoteVersionChunksBeyondFunc: func(ctx context.Context, arg db.DeleteNoteVersionChunksBeyondParams) error {
				return nil
			},
		}

		err := generatenoteversionembedding.Resolve(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err)

		chunks := mdchunk.Split(noteView.Title, noteView.Content)
		require.NotEmpty(t, chunks)
		budget := cfg.ResolvedMaxInputTokens() * 9 / 10

		// Requests: [0] whole note, [1] chunk batch.
		require.Len(t, requests, 2)
		chunkInputs := requests[1]
		require.Len(t, chunkInputs, len(chunks))
		for i, sent := range chunkInputs {
			want := mdchunk.TruncateToTokens(chunks[i].Content, budget)
			require.Less(t, len(want), len(chunks[i].Content),
				"test setup: chunk %d must exceed the budget so truncation is observable", i)
			require.Equal(t, want, sent,
				"chunk %d must be truncated to the max_input_tokens budget before embedding", i)
		}
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
