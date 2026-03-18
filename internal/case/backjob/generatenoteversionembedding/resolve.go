package generatenoteversionembedding

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg generatenoteversionembedding_test . Env

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/mdchunk"
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
	GetNoteVersionChunks(ctx context.Context, versionID int64) ([]db.NoteVersionChunk, error)
	UpsertNoteVersionChunk(ctx context.Context, arg db.UpsertNoteVersionChunkParams) error
	DeleteNoteVersionChunksBeyond(ctx context.Context, arg db.DeleteNoteVersionChunksBeyondParams) error
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

	strippedContent := mdchunk.StripFrontmatter(string(noteView.Content))

	// Calculate content hash (uses stripped content to avoid noise from frontmatter changes)
	contentHash := sha256.Sum256([]byte(noteView.Title + strippedContent))

	// Check if embedding already exists with same content hash
	existing, err := env.GetNoteVersionEmbedding(ctx, params.VersionID)
	if err == nil && bytes.Equal(existing.ContentHash, contentHash[:]) {
		env.Logger().Debug("embedding already up to date", "version_id", params.VersionID)
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing embedding: %w", err)
	}

	// Prepare text for embedding (title + stripped content)
	text := noteView.Title + "\n\n" + strippedContent

	// Generate embedding
	result, err := env.OpenAI().CreateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	// Save embedding
	err = env.UpsertNoteVersionEmbedding(ctx, db.UpsertNoteVersionEmbeddingParams{
		VersionID:   params.VersionID,
		Embedding:   model.Float32SliceToBytes(result.Vector),
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

	// Generate chunk embeddings (skip search-excluded and system notes)
	if noteView.ExcludeSearch || noteView.IsSystem() {
		return nil
	}

	if err = generateChunkEmbeddings(ctx, env, params.VersionID, noteView.Title, noteView.Content); err != nil {
		return err
	}

	return nil
}

// generateChunkEmbeddings splits the note into chunks, batch-embeds changed chunks,
// upserts them, and deletes any orphan chunks from previous versions.
func generateChunkEmbeddings(ctx context.Context, env Env, versionID int64, title string, rawContent []byte) error {
	chunks := mdchunk.Split(title, rawContent)

	// Load existing chunk hashes to skip unchanged chunks.
	existingChunks, err := env.GetNoteVersionChunks(ctx, versionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get existing chunks: %w", err)
	}
	existingHashes := make(map[int64][]byte, len(existingChunks))
	for _, ec := range existingChunks {
		existingHashes[ec.ChunkIndex] = ec.ContentHash
	}

	// Identify chunks that need re-embedding.
	type pendingChunk struct {
		chunk mdchunk.Chunk
		hash  [32]byte
	}
	var toEmbed []pendingChunk
	for _, c := range chunks {
		h := sha256.Sum256([]byte(c.Content))
		if existing, ok := existingHashes[int64(c.Index)]; ok && bytes.Equal(existing, h[:]) {
			continue // content unchanged, skip
		}
		toEmbed = append(toEmbed, pendingChunk{chunk: c, hash: h})
	}

	if len(toEmbed) > 0 {
		texts := make([]string, len(toEmbed))
		for i, pe := range toEmbed {
			texts[i] = pe.chunk.Content
		}

		results, embErr := env.OpenAI().CreateEmbeddings(ctx, texts)
		if embErr != nil {
			return fmt.Errorf("failed to create chunk embeddings: %w", embErr)
		}

		modelID := int64(env.Features().VectorSearch.Model)
		for i, pe := range toEmbed {
			tokens := int64(results[i].Tokens)
			if upsertErr := env.UpsertNoteVersionChunk(ctx, db.UpsertNoteVersionChunkParams{
				VersionID:   versionID,
				ChunkIndex:  int64(pe.chunk.Index),
				Content:     pe.chunk.Content,
				Embedding:   model.Float32SliceToBytes(results[i].Vector),
				ModelID:     &modelID,
				ContentHash: pe.hash[:],
				Tokens:      &tokens,
			}); upsertErr != nil {
				return fmt.Errorf("failed to upsert chunk %d: %w", pe.chunk.Index, upsertErr)
			}
		}
	}

	// Delete orphan chunks from a previous version that had more chunks.
	if delErr := env.DeleteNoteVersionChunksBeyond(ctx, db.DeleteNoteVersionChunksBeyondParams{
		VersionID:  versionID,
		ChunkIndex: int64(len(chunks) - 1),
	}); delErr != nil {
		return fmt.Errorf("failed to delete orphan chunks: %w", delErr)
	}

	return nil
}
