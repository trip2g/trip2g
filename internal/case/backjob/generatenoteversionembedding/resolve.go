package generatenoteversionembedding

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg generatenoteversionembedding_test . Env

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

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

	vsConfig := env.Features().VectorSearch

	// Calculate content hash (uses stripped content to avoid noise from frontmatter
	// changes; custom-model identity is mixed in, see modelFingerprint)
	contentHash := sha256.Sum256([]byte(noteView.Title + strippedContent + modelFingerprint(vsConfig)))

	// Check if embedding already exists with same content hash and model
	existing, err := env.GetNoteVersionEmbedding(ctx, params.VersionID)
	if err == nil && bytes.Equal(existing.ContentHash, contentHash[:]) && storedModelMatches(existing.ModelID, vsConfig.Model) {
		// Hash matches — skip whole-note embedding regeneration.
		// But still check if chunk embeddings are missing (e.g. first deploy after chunk feature).
		if noteView.ExcludeSearch || noteView.IsSystem() {
			env.Logger().Debug("embedding already up to date, skipping", "version_id", params.VersionID)
			return nil
		}
		existingChunks, chunkErr := env.GetNoteVersionChunks(ctx, params.VersionID)
		if chunkErr == nil && len(existingChunks) > 0 {
			env.Logger().Debug("embedding and chunks already up to date, skipping", "version_id", params.VersionID)
			return nil
		}
		env.Logger().Debug("embedding up to date but chunks missing, generating", "version_id", params.VersionID, "title", noteView.Title)
		return generateChunkEmbeddings(ctx, env, params.VersionID, noteView.Title, noteView.Content)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing embedding: %w", err)
	}
	env.Logger().Debug("generating embedding", "version_id", params.VersionID, "title", noteView.Title)

	// Prepare text for embedding (title + stripped content, with model-specific passage prefix).
	passagePrefix := vsConfig.ResolvedPassagePrefix()
	// Cap whole-note input to the model's hard token limit; notes can be larger
	// than the embedding window, which otherwise fails with HTTP 400 and retries
	// forever. estimateTokens is approximate, so leave a 10% safety margin.
	budget := vsConfig.ResolvedMaxInputTokens() * 9 / 10
	text := passagePrefix + mdchunk.TruncateToTokens(noteView.Title+"\n\n"+strippedContent, budget)

	// Generate embedding
	result, err := env.OpenAI().CreateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	// Save embedding
	err = env.UpsertNoteVersionEmbedding(ctx, db.UpsertNoteVersionEmbeddingParams{
		VersionID:   params.VersionID,
		Embedding:   model.Float32SliceToBytes(result.Vector),
		ModelID:     int64(vsConfig.Model),
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
	env.Logger().Debug("chunk split done", "version_id", versionID, "title", title, "total_chunks", len(chunks))

	// Load existing chunks to skip ones unchanged under the same model.
	existingChunks, err := env.GetNoteVersionChunks(ctx, versionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get existing chunks: %w", err)
	}
	existingByIndex := make(map[int64]db.NoteVersionChunk, len(existingChunks))
	for _, ec := range existingChunks {
		existingByIndex[ec.ChunkIndex] = ec
	}

	vsConfig := env.Features().VectorSearch
	fingerprint := modelFingerprint(vsConfig)

	// Identify chunks that need re-embedding.
	type pendingChunk struct {
		chunk mdchunk.Chunk
		hash  [32]byte
	}
	var toEmbed []pendingChunk
	for _, c := range chunks {
		h := sha256.Sum256([]byte(c.Content + fingerprint))
		if ec, ok := existingByIndex[int64(c.Index)]; ok && bytes.Equal(ec.ContentHash, h[:]) &&
			(ec.ModelID == nil || storedModelMatches(*ec.ModelID, vsConfig.Model)) {
			continue // content and model unchanged, skip
		}
		toEmbed = append(toEmbed, pendingChunk{chunk: c, hash: h})
	}

	env.Logger().Debug("chunks to embed", "version_id", versionID, "to_embed", len(toEmbed), "skipped", len(chunks)-len(toEmbed))

	if len(toEmbed) > 0 {
		passagePrefix := vsConfig.ResolvedPassagePrefix()
		texts := make([]string, len(toEmbed))
		for i, pe := range toEmbed {
			texts[i] = passagePrefix + pe.chunk.Content
		}

		results, embErr := env.OpenAI().CreateEmbeddings(ctx, texts)
		if embErr != nil {
			return fmt.Errorf("failed to create chunk embeddings: %w", embErr)
		}

		modelID := int64(vsConfig.Model)
		for i, pe := range toEmbed {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
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

		env.Logger().Debug("chunk embeddings saved", "version_id", versionID, "saved", len(toEmbed))
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

// modelFingerprint identifies the embedding model inside content hashes. Every
// custom model shares ModelID 0, so model identity has to live in the hash:
// resolved name and passage prefix change the stored vectors, dimensions change
// the vector width, and max_input_tokens changes how much of a long note gets
// embedded — all four invalidate stored rows. Dimensions and max_input_tokens
// can be overridden for built-in models too, so the fingerprint covers all
// models. base_url is deliberately excluded: moving the same model to another
// server must not invalidate every embedding (identity = name + params).
func modelFingerprint(cfg features.VectorSearchConfig) string {
	return "\x00model=" + cfg.ResolvedModelName() +
		"\x00passage_prefix=" + cfg.ResolvedPassagePrefix() +
		"\x00dimensions=" + strconv.Itoa(cfg.ResolvedDimensions()) +
		"\x00max_input_tokens=" + strconv.Itoa(cfg.ResolvedMaxInputTokens())
}

// storedModelMatches reports whether a stored row was embedded by the configured
// model. Rows written before model tracking carry ModelID 0 and are trusted;
// custom models all store ModelID 0 and are told apart via modelFingerprint.
func storedModelMatches(stored int64, configured features.EmbeddingModel) bool {
	return stored == 0 || stored == int64(configured)
}
