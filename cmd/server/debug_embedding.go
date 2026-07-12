package main

import (
	"encoding/json"
	"net/http"
	"time"

	"trip2g/internal/mdchunk"
	"trip2g/internal/openai"
)

// debugEmbeddingEnabled gates /debug/embedding registration: vector search
// must be on, and — since the endpoint is unauthenticated and can burn
// embedding API/CPU — dev mode or the explicit --debug-embedding flag too.
func (a *app) debugEmbeddingEnabled() bool {
	return a.openaiClient != nil && (a.config.DevMode || a.config.DebugEmbedding)
}

// handleDebugEmbedding provides a step-by-step view of the indexing pipeline:
// chunk splitting → embedding → response with timing and vector samples.
//
// POST /debug/embedding
//
//	{"title": "My Note", "content": "# Heading\n\nParagraph text..."}
func (a *app) handleDebugEmbedding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		req.Title = "Debug Note"
	}

	type chunkResult struct {
		Index          int       `json:"index"`
		ContentPreview string    `json:"content_preview"` // first 120 chars
		Chars          int       `json:"chars"`
		DimCount       int       `json:"dim_count,omitempty"`
		FirstDims      []float32 `json:"first_dims,omitempty"`
		EmbedErr       string    `json:"embed_err,omitempty"`
	}
	type response struct {
		Model         string        `json:"model"`
		BaseURL       string        `json:"base_url,omitempty"`
		PassagePrefix string        `json:"passage_prefix,omitempty"`
		ChunkCount    int           `json:"chunk_count"`
		Chunks        []chunkResult `json:"chunks"`
		SplitMs       int64         `json:"split_ms"`
		EmbedMs       int64         `json:"embed_ms"`
		TotalMs       int64         `json:"total_ms"`
		EmbedErr      string        `json:"embed_err,omitempty"`
	}

	t0 := time.Now()

	splitStart := time.Now()
	chunks := mdchunk.Split(req.Title, []byte(req.Content))
	splitMs := time.Since(splitStart).Milliseconds()

	results := make([]chunkResult, len(chunks))
	for i, c := range chunks {
		preview := c.Content
		if len(preview) > 120 {
			preview = preview[:120] + "…"
		}
		results[i] = chunkResult{
			Index:          c.Index,
			ContentPreview: preview,
			Chars:          len(c.Content),
		}
	}

	// Embed all chunks in one batch call (with model-specific passage prefix).
	passagePrefix := a.config.Features.VectorSearch.ResolvedPassagePrefix()
	var embedMs int64
	var embedErrStr string
	if a.openaiClient == nil { //nolint:nestif // embedding branch complexity is inherent to the debug handler
		embedErrStr = "vector search not enabled (openaiClient is nil)"
	} else {
		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = passagePrefix + c.Content
		}
		embedStart := time.Now()
		embeddings, err := a.openaiClient.CreateEmbeddings(r.Context(), texts, openai.KindDebug)
		embedMs = time.Since(embedStart).Milliseconds()
		if err != nil {
			embedErrStr = err.Error()
			for i := range results {
				results[i].EmbedErr = err.Error()
			}
		} else {
			for i, e := range embeddings {
				results[i].DimCount = len(e.Vector)
				if len(e.Vector) >= 8 {
					results[i].FirstDims = e.Vector[:8]
				} else {
					results[i].FirstDims = e.Vector
				}
			}
		}
	}

	resp := response{
		Model:         a.config.Features.VectorSearch.ModelName,
		BaseURL:       a.config.Features.VectorSearch.BaseURL,
		PassagePrefix: passagePrefix,
		ChunkCount:    len(chunks),
		Chunks:        results,
		SplitMs:       splitMs,
		EmbedMs:       embedMs,
		TotalMs:       time.Since(t0).Milliseconds(),
		EmbedErr:      embedErrStr,
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}
