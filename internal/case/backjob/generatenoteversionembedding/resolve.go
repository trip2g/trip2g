package generatenoteversionembedding

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg generatenoteversionembedding_test . Env

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/openai"
)

type Params struct {
	VersionID int64
}

type Env interface {
	Logger() logger.Logger
	Features() features.Features
	OpenAI() *openai.Client
	LatestNoteViews() *model.NoteViews
	GetNoteVersionEmbedding(ctx context.Context, versionID int64) (db.NoteVersionEmbedding, error)
	UpsertNoteVersionEmbedding(ctx context.Context, arg db.UpsertNoteVersionEmbeddingParams) error
}

func Resolve(ctx context.Context, env Env, params Params) error {
	if !env.Features().VectorSearch.Enabled {
		env.Logger().Debug("vector search disabled, skipping embedding generation")
		return nil
	}

	// Get note from in-memory cache
	noteView := env.LatestNoteViews().GetByVersionID(params.VersionID)
	if noteView == nil {
		env.Logger().Warn("note version not found in cache", "version_id", params.VersionID)
		return nil // Note might have been deleted, skip silently
	}

	// Calculate content hash
	contentHash := sha256.Sum256([]byte(noteView.Title + string(noteView.Content)))

	// Check if embedding already exists with same content hash
	existing, err := env.GetNoteVersionEmbedding(ctx, params.VersionID)
	if err == nil && bytes.Equal(existing.ContentHash, contentHash[:]) {
		env.Logger().Debug("embedding already up to date", "version_id", params.VersionID)
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing embedding: %w", err)
	}

	// Prepare text for embedding (title + content)
	text := noteView.Title + "\n\n" + string(noteView.Content)

	// Generate embedding
	result, err := env.OpenAI().CreateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	// Save embedding
	err = env.UpsertNoteVersionEmbedding(ctx, db.UpsertNoteVersionEmbeddingParams{
		VersionID:   params.VersionID,
		Embedding:   Float32SliceToBytes(result.Vector),
		ModelID:     int64(env.Features().VectorSearch.Model),
		ContentHash: contentHash[:],
		Tokens:      int64(result.Tokens),
	})
	if err != nil {
		return fmt.Errorf("failed to save embedding: %w", err)
	}

	env.Logger().Info("generated embedding",
		"version_id", params.VersionID,
		"tokens", result.Tokens,
		"dimensions", len(result.Vector),
	)

	return nil
}

// Float32SliceToBytes converts []float32 to []byte for storage.
func Float32SliceToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// BytesToFloat32Slice converts []byte back to []float32.
func BytesToFloat32Slice(data []byte) []float32 {
	floats := make([]float32, len(data)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return floats
}
