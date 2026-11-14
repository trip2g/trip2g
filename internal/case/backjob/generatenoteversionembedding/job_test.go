package generatenoteversionembedding_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/openai"

	"github.com/stretchr/testify/require"
)

type jobEnv struct {
	*EnvMock
	enqueued   []model.BackgroundTask
	enqueueErr error
}

func (e *jobEnv) RegisterJob(model.BackgroundQueueID, string, func(context.Context, []byte) error) {
}

func (e *jobEnv) EnqueueJob(_ context.Context, task model.BackgroundTask) error {
	if e.enqueueErr != nil {
		return e.enqueueErr
	}
	e.enqueued = append(e.enqueued, task)
	return nil
}

// newJobEnv serves one note through an embedding endpoint that answers HTTP
// 503 while *failing is true, and keeps what the job writes.
func newJobEnv(t *testing.T, failing *bool) *jobEnv {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *failing {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
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
	t.Cleanup(srv.Close)

	cfg := features.VectorSearchConfig{Enabled: true, Model: features.EmbeddingModelSmall}
	noteView := &model.NoteView{VersionID: 1, Title: "Test Note", Content: []byte("Test content"), Permalink: "/test-note"}

	var embedding *db.NoteVersionEmbedding
	var chunks []db.NoteVersionChunk

	return &jobEnv{EnvMock: &EnvMock{
		NoteEmbeddingsSavedFunc: func() {},
		FeaturesFunc:            func() features.Features { return features.Features{VectorSearch: cfg} },
		LoggerFunc:              func() logger.Logger { return &logger.TestLogger{} },
		LatestNoteViewsFunc: func() *model.NoteViews {
			return &model.NoteViews{Map: map[string]*model.NoteView{noteView.Permalink: noteView}}
		},
		OpenAIFunc: func() *openai.Client {
			return openai.New("test-key", "test-model", srv.URL+"/v1")
		},
		GetNoteVersionEmbeddingFunc: func(ctx context.Context, versionID int64) (db.NoteVersionEmbedding, error) {
			if embedding == nil {
				return db.NoteVersionEmbedding{}, sql.ErrNoRows
			}
			return *embedding, nil
		},
		UpsertNoteVersionEmbeddingFunc: func(ctx context.Context, arg db.UpsertNoteVersionEmbeddingParams) error {
			embedding = &db.NoteVersionEmbedding{VersionID: arg.VersionID, ModelID: arg.ModelID, ContentHash: arg.ContentHash}
			return nil
		},
		GetNoteVersionChunksFunc: func(ctx context.Context, versionID int64) ([]db.NoteVersionChunk, error) {
			return chunks, nil
		},
		UpsertNoteVersionChunkFunc: func(ctx context.Context, arg db.UpsertNoteVersionChunkParams) error {
			chunks = append(chunks, db.NoteVersionChunk{
				VersionID: arg.VersionID, ChunkIndex: arg.ChunkIndex, ModelID: arg.ModelID, ContentHash: arg.ContentHash,
			})
			return nil
		},
		DeleteNoteVersionChunksBeyondFunc: func(ctx context.Context, arg db.DeleteNoteVersionChunksBeyondParams) error {
			return nil
		},
	}}
}

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("a failed version is embedded by its delayed retry", func(t *testing.T) {
		failing := true
		env := newJobEnv(t, &failing)

		err := generatenoteversionembedding.Run(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err, "a scheduled retry finishes the failed job")
		require.Equal(t, []model.BackgroundTask{{
			ID:       generatenoteversionembedding.JobID,
			Queue:    generatenoteversionembedding.QueueID,
			Data:     generatenoteversionembedding.Params{VersionID: 1, Attempt: 1},
			Priority: generatenoteversionembedding.Priority,
			Delay:    time.Minute,
		}}, env.enqueued)
		require.Empty(t, env.UpsertNoteVersionEmbeddingCalls())

		failing = false
		retry, ok := env.enqueued[0].Data.(generatenoteversionembedding.Params)
		require.True(t, ok)

		err = generatenoteversionembedding.Run(ctx, env, retry)
		require.NoError(t, err)
		require.Len(t, env.enqueued, 1, "a successful retry schedules nothing")
		require.Len(t, env.UpsertNoteVersionEmbeddingCalls(), 1)
		require.NotEmpty(t, env.UpsertNoteVersionChunkCalls())
	})

	t.Run("a retry of an embedded unchanged version calls no endpoint", func(t *testing.T) {
		failing := false
		env := newJobEnv(t, &failing)

		err := generatenoteversionembedding.Run(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.NoError(t, err)
		require.Len(t, env.UpsertNoteVersionEmbeddingCalls(), 1)
		chunkWrites := len(env.UpsertNoteVersionChunkCalls())

		failing = true
		err = generatenoteversionembedding.Run(ctx, env, generatenoteversionembedding.Params{VersionID: 1, Attempt: 3})
		require.NoError(t, err, "an up-to-date version must not reach the failing endpoint")
		require.Len(t, env.UpsertNoteVersionEmbeddingCalls(), 1)
		require.Len(t, env.UpsertNoteVersionChunkCalls(), chunkWrites)
		require.Empty(t, env.enqueued)
	})

	t.Run("the last attempt schedules no retry", func(t *testing.T) {
		failing := true
		env := newJobEnv(t, &failing)

		last := generatenoteversionembedding.Params{VersionID: 1, Attempt: generatenoteversionembedding.MaxAttempts - 1}
		err := generatenoteversionembedding.Run(ctx, env, last)
		require.Error(t, err)
		require.Empty(t, env.enqueued)
	})

	t.Run("a retry that cannot be scheduled fails the job", func(t *testing.T) {
		failing := true
		env := newJobEnv(t, &failing)
		env.enqueueErr = errors.New("queue unavailable")

		err := generatenoteversionembedding.Run(ctx, env, generatenoteversionembedding.Params{VersionID: 1})
		require.ErrorContains(t, err, "queue unavailable")
	})

	t.Run("one version is tried at most MaxAttempts times with one retry waiting", func(t *testing.T) {
		failing := true
		env := newJobEnv(t, &failing)

		params := generatenoteversionembedding.Params{VersionID: 1}
		runs := 0
		for {
			runs++
			err := generatenoteversionembedding.Run(ctx, env, params)
			if err != nil {
				break
			}
			require.Len(t, env.enqueued, runs, "each failure schedules exactly one retry")
			next, ok := env.enqueued[len(env.enqueued)-1].Data.(generatenoteversionembedding.Params)
			require.True(t, ok)
			params = next
		}
		require.Equal(t, generatenoteversionembedding.MaxAttempts, runs)
	})
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, time.Minute},
		{1, 2 * time.Minute},
		{2, 4 * time.Minute},
		{5, 32 * time.Minute},
		{6, time.Hour},
		{generatenoteversionembedding.MaxAttempts, time.Hour},
		{1000, time.Hour},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, generatenoteversionembedding.RetryDelay(tc.attempt), "attempt %d", tc.attempt)
	}
}
