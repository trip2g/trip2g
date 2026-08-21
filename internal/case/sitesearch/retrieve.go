package sitesearch

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/openai"
	"trip2g/internal/reranker"

	appmodel "trip2g/internal/model"
)

// RetrieveEnv declares the dependencies of the shared retrieval core. It is a
// subset of Env, and the MCP adapter's Env satisfies it too.
type RetrieveEnv interface {
	SearchLatestNotes(query string) ([]appmodel.SearchResult, error)
	SearchLiveNotes(query string) ([]appmodel.SearchResult, error)
	Features() features.Features
	OpenAI() *openai.Client
	LatestNoteViews() *appmodel.NoteViews
	LiveNoteViews() *appmodel.NoteViews
	LatestNoteChunks() []appmodel.NoteChunk
	LiveNoteChunks() []appmodel.NoteChunk
	Logger() logger.Logger
}

// rrfK is the RRF rank constant. Higher values reduce the impact of top ranks.
// Standard value is 60.
const rrfK = 60

// vectorTopK is the number of unique-note vector candidates fed into RRF fusion.
// Keep this wide: the cosine scan already scores every chunk, so truncating
// before fusion only discards recall at zero compute saving. The final result
// list is capped after permission filtering (see hybridResultCap in Resolve).
const vectorTopK = 50

// Retrieve runs the shared first-stage retrieval pipeline used by both the
// site (GraphQL) search and the MCP search tool: text lane (bleve) + vector
// lane + RRF fusion + optional blended cross-encoder rerank. useLatest selects
// the corpus (latest = drafts included, live = published only). merged reports
// whether the vector lane contributed, so callers can apply hybrid-only caps.
// Permission filtering and output shaping stay with the callers.
//
// rerank carries the caller's explicit preference for the second stage: nil
// means "no preference", and the instance's reranker.default decides. It can
// never turn on a reranker that is not configured.
func Retrieve(
	ctx context.Context,
	env RetrieveEnv,
	query string,
	useLatest bool,
	rerank *bool,
) ([]appmodel.SearchResult, bool, error) {
	var results []appmodel.SearchResult
	var err error
	if useLatest {
		results, err = env.SearchLatestNotes(query)
		if err != nil {
			return nil, false, fmt.Errorf("failed to SearchLatestNotes: %w", err)
		}
	} else {
		results, err = env.SearchLiveNotes(query)
		if err != nil {
			return nil, false, fmt.Errorf("failed to SearchLiveNotes: %w", err)
		}
	}

	// Mark text results
	for i := range results {
		results[i].MatchOrigin = appmodel.SearchMatchText
	}

	// Hybrid search: add vector results if enabled
	// passageByURL holds the best-matching chunk passage per note (window-sized),
	// used by the optional reranker to avoid feeding truncated whole notes.
	var passageByURL map[string]string
	merged := false
	if env.Features().VectorSearch.Enabled && env.OpenAI() != nil {
		vectorResults, passages, vectorErr := vectorSearch(ctx, env, query, useLatest)
		if vectorErr != nil {
			// Log error but don't fail - text search still works
			env.Logger().Warn("vector search failed", "error", vectorErr)
		} else {
			passageByURL = passages
			results = mergeResults(results, vectorResults)
			merged = true
		}
	}

	// Optional second-stage cross-encoder rerank, BLENDED with the RRF order.
	results = rerankResults(ctx, env, query, results, passageByURL, rerank)

	return results, merged, nil
}

func vectorSearch(ctx context.Context, env RetrieveEnv, query string, useLatest bool) ([]appmodel.SearchResult, map[string]string, error) {
	queryPrefix := env.Features().VectorSearch.ResolvedQueryPrefix()
	embedding, err := env.OpenAI().CreateEmbedding(ctx, queryPrefix+query, openai.KindQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	var chunks []appmodel.NoteChunk
	if useLatest {
		chunks = env.LatestNoteChunks()
	} else {
		chunks = env.LiveNoteChunks()
	}

	type scored struct {
		path  string
		chunk appmodel.NoteChunk
		sim   float64
	}

	// Score all chunks, no absolute threshold — E5 models compress scores
	// into 0.7–1.0 range, making absolute thresholds unreliable.
	// dotSimilarity is used here instead of cosine because the embedding server
	// returns L2-normalised unit vectors (TEI always normalizes /v1/embeddings output),
	// making cosine ≡ dot product at lower compute cost.
	scanStart := time.Now()
	var candidates []scored
	mismatched := 0
	for _, c := range chunks {
		// A stored vector with a different dimensionality (model switch without
		// re-embedding) is incomparable — skip it rather than scoring it 0, which
		// would surface an arbitrary alphabetical top-K.
		if len(c.Embedding) != len(embedding.Vector) {
			mismatched++
			continue
		}
		sim := dotSimilarity(embedding.Vector, c.Embedding)
		candidates = append(candidates, scored{c.NotePath, c, sim})
	}
	if mismatched > 0 {
		env.Logger().Warn("vector search: embedding dimension mismatch, skipping stale chunks (model switch without re-embedding?)",
			"query_dims", len(embedding.Vector), "skipped_chunks", mismatched)
	}
	env.Logger().Debug("vector scan complete", "chunks", len(chunks), "duration", time.Since(scanStart))

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].sim != candidates[j].sim {
			return candidates[i].sim > candidates[j].sim
		}
		return candidates[i].path < candidates[j].path
	})

	var noteViews *appmodel.NoteViews
	if useLatest {
		noteViews = env.LatestNoteViews()
	} else {
		noteViews = env.LiveNoteViews()
	}

	// Deduplicate by note path, take top-K unique notes.
	// passageByURL records the highest-similarity chunk per note — a window-sized
	// passage (chunks are capped ~450 tokens, under the CE ~512-token window) so
	// the reranker never sees a truncated whole note.
	seen := map[string]bool{}
	passageByURL := make(map[string]string)
	var results []appmodel.SearchResult
	for _, c := range candidates {
		if seen[c.path] {
			continue
		}
		seen[c.path] = true
		note := noteViews.PathMap[c.path]
		if note == nil {
			continue
		}
		title := note.Title
		chunkIndex := c.chunk.ChunkIndex
		results = append(results, appmodel.SearchResult{
			NoteView:           note,
			URL:                note.Permalink,
			Score:              c.sim,
			MatchOrigin:        appmodel.SearchMatchVector,
			HighlightedTitle:   &title,
			HighlightedContent: []string{SnippetFromChunk(c.chunk.Content, 200)},
			ChunkIndex:         &chunkIndex,
		})
		passageByURL[note.Permalink] = chunkPassage(c.chunk.Content)
		if len(results) >= vectorTopK {
			break
		}
	}

	return results, passageByURL, nil
}

// chunkPassage strips the breadcrumb prefix ("{title} > {h1} > {h2}\n\n") from a
// chunk, returning the body text used as the reranker document. The body is
// already window-sized (chunks are capped ~450 tokens at ingest).
func chunkPassage(content string) string {
	if idx := strings.Index(content, "\n\n"); idx >= 0 {
		content = content[idx+2:]
	}
	return TrimWhitespace(content)
}

// SnippetFromChunk extracts a display snippet from chunk content.
// Chunks have the format "{title} > {h1} > {h2}\n\n{body}", so we skip past the breadcrumb prefix.
func SnippetFromChunk(content string, maxLen int) string {
	if idx := strings.Index(content, "\n\n"); idx >= 0 {
		content = content[idx+2:]
	}
	content = TrimWhitespace(content)
	runes := []rune(content)
	if len(runes) > maxLen {
		content = string(runes[:maxLen])
		if lastSpace := lastIndexByte(content, ' '); lastSpace > maxLen/2 {
			content = content[:lastSpace]
		}
		content += "..."
	}
	return content
}

// mergeResults combines text and vector search results using Reciprocal Rank Fusion (RRF).
// RRF score = Σ 1/(k + rank) across all result lists, using only ranks not raw scores.
// This avoids score normalization issues when combining BM25 and cosine similarity scores.
func mergeResults(textResults, vectorResults []appmodel.SearchResult) []appmodel.SearchResult {
	if len(vectorResults) == 0 {
		return textResults
	}

	type entry struct {
		result   appmodel.SearchResult
		rrfScore float64
	}

	resultMap := make(map[string]*entry)

	// Add text results with rank-based RRF score (1-indexed ranks)
	for rank, r := range textResults {
		score := 1.0 / float64(rrfK+rank+1)
		if e, ok := resultMap[r.URL]; ok {
			e.rrfScore += score
		} else {
			resultMap[r.URL] = &entry{result: r, rrfScore: score}
		}
	}

	// Add vector results with rank-based RRF score
	for rank, r := range vectorResults {
		score := 1.0 / float64(rrfK+rank+1)
		if e, ok := resultMap[r.URL]; ok {
			// Note exists in text results too — accumulate score, mark as hybrid,
			// and inherit the vector lane's chunk pointer (the text lane has none).
			e.rrfScore += score
			e.result.MatchOrigin = appmodel.SearchMatchHybrid
			if e.result.ChunkIndex == nil && r.ChunkIndex != nil {
				e.result.ChunkIndex = r.ChunkIndex
			}
			if len(e.result.HighlightedContent) == 0 && len(r.HighlightedContent) > 0 {
				e.result.HighlightedContent = r.HighlightedContent
			}
		} else {
			// Vector-only result — use chunk snippet if pre-set, else fall back to note snippet
			if len(r.HighlightedContent) == 0 {
				title := r.NoteView.Title
				r.HighlightedTitle = &title
				r.HighlightedContent = []string{generateSnippet(r.NoteView, 150)}
			}
			resultMap[r.URL] = &entry{result: r, rrfScore: score}
		}
	}

	finalResults := make([]appmodel.SearchResult, 0, len(resultMap))
	for _, e := range resultMap {
		e.result.Score = e.rrfScore
		finalResults = append(finalResults, e.result)
	}

	sort.Slice(finalResults, func(i, j int) bool {
		if finalResults[i].Score != finalResults[j].Score {
			return finalResults[i].Score > finalResults[j].Score
		}
		return finalResults[i].URL < finalResults[j].URL
	})

	return finalResults
}

// rerankResults applies the shared optional cross-encoder rerank
// (reranker.BlendRRF) to the fused candidate set, blending the CE score with
// the first-stage RRF order. No-op when the reranker is not configured, or when
// this request did not ask for it and the instance does not rerank by default.
// See docs/dev/reranker.md.
func rerankResults(
	ctx context.Context,
	env RetrieveEnv,
	query string,
	results []appmodel.SearchResult,
	passageByURL map[string]string,
	want *bool,
) []appmodel.SearchResult {
	cfg := env.Features().VectorSearch.Reranker
	if !cfg.ShouldRerank(want) {
		return results
	}
	return reranker.BlendRRF(ctx, cfg, env.Logger(), query, results, passageByURL)
}

// generateSnippet extracts a text snippet from note content for vector-only results.
func generateSnippet(note *appmodel.NoteView, maxLen int) string {
	// Use plain text content if available
	text := string(note.Content)
	if len(text) == 0 {
		return ""
	}

	// Skip frontmatter if present
	if len(text) > 3 && text[:3] == "---" {
		if idx := findSecondFrontmatter(text); idx > 0 {
			text = text[idx+3:]
		}
	}

	// Trim and limit length
	text = TrimWhitespace(text)
	runes := []rune(text)
	if len(runes) > maxLen {
		text = string(runes[:maxLen])
		if lastSpace := lastIndexByte(text, ' '); lastSpace > maxLen/2 {
			text = text[:lastSpace]
		}
		text += "..."
	}

	return text
}

func findSecondFrontmatter(s string) int {
	// Find closing --- after the opening ---
	for i := 4; i < len(s)-2; i++ {
		if s[i] == '-' && s[i+1] == '-' && s[i+2] == '-' {
			return i
		}
	}
	return -1
}

// TrimWhitespace trims leading/trailing whitespace and collapses internal runs
// of whitespace to a single space.
func TrimWhitespace(s string) string {
	result := make([]byte, 0, len(s))
	inWhitespace := true
	for i := range len(s) {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !inWhitespace && len(result) > 0 {
				result = append(result, ' ')
				inWhitespace = true
			}
		} else {
			result = append(result, c)
			inWhitespace = false
		}
	}
	// Trim trailing space
	if len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return string(result)
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// dotSimilarity returns the dot product of two vectors.
// This is equivalent to cosine similarity when both vectors are L2-normalised,
// which is guaranteed by the embedding server (TEI always L2-normalises its
// /v1/embeddings output). Using dot product avoids the redundant sqrt
// divisions that cosine similarity would otherwise perform on unit vectors.
func dotSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}
