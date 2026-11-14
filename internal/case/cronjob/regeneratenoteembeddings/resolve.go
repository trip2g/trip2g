package regeneratenoteembeddings

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg regeneratenoteembeddings_test . Env

import (
	"context"
	"database/sql"
	"errors"

	"trip2g/internal/case/backjob/generatenoteversionembedding"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	"trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	Features() features.Features
	LatestNoteViews() *model.NoteViews
	GetNoteVersionEmbeddingsByVersionIDs(ctx context.Context, versionIDs []int64) ([]db.NoteVersionEmbedding, error)
	GetNoteVersionChunkHashesByVersionIDs(ctx context.Context, versionIDs []int64) ([]db.GetNoteVersionChunkHashesByVersionIDsRow, error)
	EnqueueGenerateNoteVersionEmbedding(ctx context.Context, versionID int64) error
}

type Result struct {
	TotalNotes    int
	EnqueuedCount int
	UpToDateCount int
	Errors        []error
}

func Resolve(ctx context.Context, env Env) (*Result, error) {
	return resolveAndRecord(ctx, env, false)
}

// ResolveForced enqueues every note regardless of its stored content hash. The
// hash covers title, content and the model fingerprint, so a change to how a
// note is split into chunks does not move it and the cron would report every
// note as up to date forever. The per-chunk hashes still decide what is
// actually re-embedded, so unaffected chunks cost nothing.
func ResolveForced(ctx context.Context, env Env) (*Result, error) {
	return resolveAndRecord(ctx, env, true)
}

func resolveAndRecord(ctx context.Context, env Env, force bool) (*Result, error) {
	result, err := resolve(ctx, env, force)
	if result != nil {
		metrics.EmbeddingMetricsFromContext(ctx).RecordRegen(result.EnqueuedCount, result.UpToDateCount, len(result.Errors))
	}
	return result, err
}

func resolve(ctx context.Context, env Env, force bool) (*Result, error) {
	result := &Result{}

	if !env.Features().VectorSearch.Enabled {
		env.Logger().Debug("vector search disabled, skipping embedding regeneration")
		return result, nil
	}

	noteViews := env.LatestNoteViews()
	result.TotalNotes = len(noteViews.List)

	if result.TotalNotes == 0 {
		return result, nil
	}

	// Collect all version IDs
	versionIDs := make([]int64, 0, len(noteViews.List))
	for _, note := range noteViews.List {
		versionIDs = append(versionIDs, note.VersionID)
	}

	// Fetch all existing embeddings
	embeddings, err := env.GetNoteVersionEmbeddingsByVersionIDs(ctx, versionIDs)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	embeddingByVersion := make(map[int64]*db.NoteVersionEmbedding, len(embeddings))
	for i := range embeddings {
		embeddingByVersion[embeddings[i].VersionID] = &embeddings[i]
	}

	chunks, err := env.GetNoteVersionChunkHashesByVersionIDs(ctx, versionIDs)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	chunksByVersion := make(map[int64][]db.GetNoteVersionChunkHashesByVersionIDsRow)
	for _, c := range chunks {
		chunksByVersion[c.VersionID] = append(chunksByVersion[c.VersionID], c)
	}

	// A note counts as up to date only when the generator would not call the
	// embedding endpoint for it: whole-note row and every chunk current. A job
	// that failed after saving the whole-note row leaves chunks missing, and a
	// hash-only check would call that note up to date forever.
	vsConfig := env.Features().VectorSearch
	for _, note := range noteViews.List {
		if !force && generatenoteversionembedding.UpToDate(note, embeddingByVersion[note.VersionID], chunksByVersion[note.VersionID], vsConfig) {
			result.UpToDateCount++
			continue
		}

		err = env.EnqueueGenerateNoteVersionEmbedding(ctx, note.VersionID)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		result.EnqueuedCount++
	}

	env.Logger().Info("embedding regeneration complete",
		"total", result.TotalNotes,
		"enqueued", result.EnqueuedCount,
		"up_to_date", result.UpToDateCount,
		"errors", len(result.Errors),
	)

	return result, nil
}
