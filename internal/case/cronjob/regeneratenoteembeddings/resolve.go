package regeneratenoteembeddings

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"

	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	Features() features.Features
	LatestNoteViews() *model.NoteViews
	GetNoteVersionEmbeddingsByVersionIDs(ctx context.Context, versionIDs []int64) ([]db.NoteVersionEmbedding, error)
	EnqueueGenerateNoteVersionEmbedding(ctx context.Context, versionID int64) error
}

type Result struct {
	TotalNotes    int
	EnqueuedCount int
	UpToDateCount int
	Errors        []error
}

func Resolve(ctx context.Context, env Env) (*Result, error) {
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

	// Build map of version_id -> content_hash
	embeddingHashes := make(map[int64][]byte, len(embeddings))
	for _, emb := range embeddings {
		embeddingHashes[emb.VersionID] = emb.ContentHash
	}

	// Check each note and enqueue if needed
	for _, note := range noteViews.List {
		currentHash := sha256.Sum256([]byte(note.Title + string(note.Content)))

		existingHash, hasEmbedding := embeddingHashes[note.VersionID]
		if hasEmbedding && bytesEqual(existingHash, currentHash[:]) {
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

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
