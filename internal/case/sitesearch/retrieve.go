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
	SearchLatestNotes(query string, from, size int) ([]appmodel.SearchResult, error)
	SearchLiveNotes(query string, from, size int) ([]appmodel.SearchResult, error)
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

// textLaneSize is the number of text-lane candidates fed into fusion, and the
// page size the text lane reads the index in.
const textLaneSize = 20

// keepScanLimit bounds how many candidates each lane checks against
// Request.Keep, so a query whose matches are mostly rejected cannot walk the
// whole index.
const keepScanLimit = 200

// exactNameWeight is the RRF weight of the exact-name list. A query that spells
// out a note's title or alias almost always means that note, so this list
// counts double against text and vector.
const exactNameWeight = 2.0

// Request is one retrieval. UseLatest selects the corpus (latest = drafts
// included, live = published only). Rerank carries the caller's explicit
// preference for the second stage: nil means "no preference", and the
// instance's reranker.default decides; it can never turn on a reranker that is
// not configured. Keep, when set, drops candidates inside the text and vector
// lanes, before their caps, so rejected notes do not take the slots of the
// ones after them.
type Request struct {
	Query     string
	UseLatest bool
	Rerank    *bool
	Keep      func(*appmodel.NoteView) (bool, error)
}

// Retrieve runs the shared first-stage retrieval pipeline used by both the
// site (GraphQL) search and the MCP search tool: text lane (bleve) + vector
// lane + RRF fusion + optional blended cross-encoder rerank. merged reports
// whether the vector lane contributed, so callers can apply hybrid-only caps.
// Permission filtering and output shaping stay with the callers.
func Retrieve(ctx context.Context, env RetrieveEnv, req Request) ([]appmodel.SearchResult, bool, error) {
	results, err := textLane(env, req)
	if err != nil {
		return nil, false, err
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
		candidates, vectorErr := vectorCandidates(ctx, env, req)
		if vectorErr != nil {
			// Log error but don't fail - text search still works
			env.Logger().Warn("vector search failed", "error", vectorErr)
		} else {
			vectorResults, passages, laneErr := vectorLane(candidates, noteViewsOf(env, req.UseLatest), req.Keep)
			if laneErr != nil {
				return nil, false, laneErr
			}
			passageByURL = passages
			results = mergeResults(results, vectorResults)
			merged = true
		}
	}

	if names := exactNameResults(req.Query, noteViewsOf(env, req.UseLatest)); len(names) > 0 {
		if !merged {
			results = rankScores(results)
		}
		results = fuseLane(results, names, exactNameWeight)
	}

	// Optional second-stage cross-encoder rerank, BLENDED with the RRF order.
	results = rerankResults(ctx, env, req.Query, results, passageByURL, req.Rerank)

	return results, merged, nil
}

func noteViewsOf(env RetrieveEnv, useLatest bool) *appmodel.NoteViews {
	if useLatest {
		return env.LatestNoteViews()
	}
	return env.LiveNoteViews()
}

// textLane returns the first textLaneSize text hits that req.Keep accepts,
// reading the index page by page up to keepScanLimit hits.
func textLane(env RetrieveEnv, req Request) ([]appmodel.SearchResult, error) {
	search := env.SearchLiveNotes
	if req.UseLatest {
		search = env.SearchLatestNotes
	}

	if req.Keep == nil {
		results, err := search(req.Query, 0, textLaneSize)
		if err != nil {
			return nil, fmt.Errorf("failed to search notes: %w", err)
		}
		return results, nil
	}

	// A short page does not mean the index is exhausted: the loader drops hits
	// whose note is gone, so only an empty page ends the scan.
	var kept []appmodel.SearchResult
	seen := map[string]bool{}
	for from := 0; from < keepScanLimit; from += textLaneSize {
		page, err := search(req.Query, from, textLaneSize)
		if err != nil {
			return nil, fmt.Errorf("failed to search notes: %w", err)
		}
		if len(page) == 0 {
			break
		}

		for _, r := range page {
			if seen[r.URL] {
				continue
			}
			seen[r.URL] = true
			ok, keepErr := req.Keep(r.NoteView)
			if keepErr != nil {
				return nil, keepErr
			}
			if !ok {
				continue
			}
			kept = append(kept, r)
			if len(kept) == textLaneSize {
				return kept, nil
			}
		}
	}

	return kept, nil
}

type vectorCandidate struct {
	path  string
	chunk appmodel.NoteChunk
	sim   float64
}

// vectorCandidates scores every chunk against the query, best first.
func vectorCandidates(ctx context.Context, env RetrieveEnv, req Request) ([]vectorCandidate, error) {
	queryPrefix := env.Features().VectorSearch.ResolvedQueryPrefix()
	embedding, err := env.OpenAI().CreateEmbedding(ctx, queryPrefix+req.Query, openai.KindQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	var chunks []appmodel.NoteChunk
	if req.UseLatest {
		chunks = env.LatestNoteChunks()
	} else {
		chunks = env.LiveNoteChunks()
	}

	// Score all chunks, no absolute threshold — E5 models compress scores
	// into 0.7–1.0 range, making absolute thresholds unreliable.
	// dotSimilarity is used here instead of cosine because the embedding server
	// returns L2-normalised unit vectors (TEI always normalizes /v1/embeddings output),
	// making cosine ≡ dot product at lower compute cost.
	scanStart := time.Now()
	var candidates []vectorCandidate
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
		candidates = append(candidates, vectorCandidate{c.NotePath, c, sim})
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

	return candidates, nil
}

// vectorLane deduplicates candidates by note path and takes the top vectorTopK
// unique notes that keep accepts, checking at most keepScanLimit of them.
func vectorLane(
	candidates []vectorCandidate,
	noteViews *appmodel.NoteViews,
	keep func(*appmodel.NoteView) (bool, error),
) ([]appmodel.SearchResult, map[string]string, error) {
	// Deduplicate by note path, take top-K unique notes.
	// passageByURL records the highest-similarity chunk per note — a window-sized
	// passage (chunks are capped ~450 tokens, under the CE ~512-token window) so
	// the reranker never sees a truncated whole note.
	seen := map[string]bool{}
	passageByURL := make(map[string]string)
	var results []appmodel.SearchResult
	checked := 0
	for _, c := range candidates {
		if seen[c.path] {
			continue
		}
		seen[c.path] = true
		note := noteViews.PathMap[c.path]
		if note == nil {
			continue
		}
		if keep != nil {
			if checked == keepScanLimit {
				break
			}
			checked++
			ok, err := keep(note)
			if err != nil {
				return nil, nil, err
			}
			if !ok {
				continue
			}
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

// exactNameResults returns the notes whose title or alias equals the query
// after NormalizeName, in NameMap order.
func exactNameResults(query string, noteViews *appmodel.NoteViews) []appmodel.SearchResult {
	if noteViews == nil {
		return nil
	}
	var results []appmodel.SearchResult
	for _, note := range noteViews.NameMap[appmodel.NormalizeName(query)] {
		if note.ExcludeSearch || note.IsSystem() {
			continue
		}
		title := note.Title
		results = append(results, appmodel.SearchResult{
			NoteView:         note,
			URL:              note.Permalink,
			MatchOrigin:      appmodel.SearchMatchText,
			HighlightedTitle: &title,
		})
	}
	return results
}

// rankScores replaces raw text-lane scores with their RRF contribution so a
// text-only result list can be fused with another lane.
func rankScores(results []appmodel.SearchResult) []appmodel.SearchResult {
	for i := range results {
		results[i].Score = 1.0 / float64(rrfK+i+1)
	}
	return results
}

// fuseLane adds a weighted RRF list to results that already carry RRF scores.
func fuseLane(results, lane []appmodel.SearchResult, weight float64) []appmodel.SearchResult {
	index := make(map[string]int, len(results))
	for i, r := range results {
		index[r.URL] = i
	}
	for rank, r := range lane {
		score := weight / float64(rrfK+rank+1)
		if i, ok := index[r.URL]; ok {
			results[i].Score += score
			continue
		}
		r.Score = score
		r.HighlightedContent = []string{generateSnippet(r.NoteView, 150)}
		index[r.URL] = len(results)
		results = append(results, r)
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].URL < results[j].URL
	})
	return results
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
