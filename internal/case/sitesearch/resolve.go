package sitesearch

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/features"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/openai"
	"trip2g/internal/reranker"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"

	appmodel "trip2g/internal/model"
)

type Env interface {
	SearchLatestNotes(query string) ([]appmodel.SearchResult, error)
	SearchLiveNotes(query string) ([]appmodel.SearchResult, error)
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	CanReadNote(ctx context.Context, note *appmodel.NoteView) (bool, error)
	SiteConfig(ctx context.Context) appmodel.SiteConfig
	Logger() logger.Logger

	// For hybrid search
	Features() features.Features
	OpenAI() *openai.Client
	LatestNoteViews() *appmodel.NoteViews
	LiveNoteViews() *appmodel.NoteViews
	LatestNoteChunks() []appmodel.NoteChunk
	LiveNoteChunks() []appmodel.NoteChunk
}

// rrfK is the RRF rank constant. Higher values reduce the impact of top ranks.
// Standard value is 60.
const rrfK = 60

func Resolve(ctx context.Context, env Env, input model.SearchInput) (*model.SearchConnection, error) {
	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	siteConfig := env.SiteConfig(ctx)

	useLatest := siteConfig.ShowDraftVersions || userToken.IsAdmin()

	var results []appmodel.SearchResult

	// Text search (bleve)
	if useLatest {
		results, err = env.SearchLatestNotes(input.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to SearchLatestNotes: %w", err)
		}
	} else {
		results, err = env.SearchLiveNotes(input.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to SearchLiveNotes: %w", err)
		}
	}

	// Mark text results
	for i := range results {
		results[i].MatchOrigin = appmodel.SearchMatchText
	}

	// Hybrid search: add vector results if enabled
	if env.Features().VectorSearch.Enabled && env.OpenAI() != nil {
		vectorResults, vectorErr := vectorSearch(ctx, env, input.Query, useLatest)
		if vectorErr != nil {
			// Log error but don't fail - text search still works
			env.Logger().Warn("vector search failed", "error", vectorErr)
		} else {
			results = mergeResults(results, vectorResults)
		}
	}

	// Second-stage cross-encoder rerank of the fused candidates (F3, optional).
	results = rerankResults(ctx, env, input.Query, results)

	// Filter results based on permissions
	conn := model.SearchConnection{}
	hiddenResults := []appmodel.SearchResult{}

	for _, res := range results {
		if res.NoteView != nil {
			// Fail-closed: scoped shortapitoken → enforce read_patterns strictly.
			// Empty patterns + scoped = deny-all (not "no restriction").
			if appreq.Scoped(ctx) {
				rp := appreq.WebhookReadPatterns(ctx)
				if len(rp) == 0 || !webhookutil.MatchesAny(res.NoteView.Path, rp) {
					continue
				}
			}

			if res.NoteView.IsSystem() || res.NoteView.ExcludeSearch {
				continue
			}

			canRead, readErr := env.CanReadNote(ctx, res.NoteView)
			if readErr != nil {
				return nil, fmt.Errorf("failed to check CanReadNote: %w", readErr)
			}

			if canRead {
				conn.Nodes = append(conn.Nodes, res)
				continue
			}

			croppedResult := appmodel.SearchResult{
				HighlightedTitle:   res.HighlightedTitle,
				URL:                res.URL,
				HighlightedContent: []string{"Закрытый материал."},
			}

			hiddenResults = append(hiddenResults, croppedResult)
		}
	}

	// Push hidden results to the end of the list
	conn.Nodes = append(conn.Nodes, hiddenResults...)

	return &conn, nil
}

// vectorTopK is the number of unique-note vector candidates fed into RRF fusion.
// Keep this wide: the cosine scan already scores every chunk, so truncating
// before fusion only discards recall at zero compute saving. The final result
// list is capped after merge (see mergeResults).
const vectorTopK = 50

func vectorSearch(ctx context.Context, env Env, query string, useLatest bool) ([]appmodel.SearchResult, error) {
	queryPrefix := env.Features().VectorSearch.Model.QueryPrefix()
	embedding, err := env.OpenAI().CreateEmbedding(ctx, queryPrefix+query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
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
	// returns L2-normalised unit vectors (embedding-server/server.py normalize_embeddings=True),
	// making cosine ≡ dot product at lower compute cost.
	scanStart := time.Now()
	var candidates []scored
	for _, c := range chunks {
		sim := dotSimilarity(embedding.Vector, c.Embedding)
		candidates = append(candidates, scored{c.NotePath, c, sim})
	}
	env.Logger().Warn("vector scan complete", "chunks", len(chunks), "duration", time.Since(scanStart))

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
	seen := map[string]bool{}
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
		results = append(results, appmodel.SearchResult{
			NoteView:           note,
			URL:                note.Permalink,
			Score:              c.sim,
			MatchOrigin:        appmodel.SearchMatchVector,
			HighlightedTitle:   &title,
			HighlightedContent: []string{snippetFromChunk(c.chunk.Content, 200)},
		})
		if len(results) >= vectorTopK {
			break
		}
	}

	return results, nil
}

// snippetFromChunk extracts a display snippet from chunk content.
// Chunks have the format "{title} > {h1} > {h2}\n\n{body}", so we skip past the breadcrumb prefix.
func snippetFromChunk(content string, maxLen int) string {
	if idx := strings.Index(content, "\n\n"); idx >= 0 {
		content = content[idx+2:]
	}
	content = trimWhitespace(content)
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

// rerankResults applies an optional cross-encoder rerank to the fused candidate
// set (F3). It reorders the top-N results by query-document relevance and keeps
// OutputK. On any error it returns the input unchanged (graceful degradation),
// matching how a failing vector lane degrades to text-only.
func rerankResults(ctx context.Context, env Env, query string, results []appmodel.SearchResult) []appmodel.SearchResult {
	cfg := env.Features().VectorSearch.Reranker
	if !cfg.Enabled || len(results) < 2 {
		return results
	}

	n := cfg.TopN
	if n > len(results) {
		n = len(results)
	}
	head := results[:n]

	docs := make([]string, len(head))
	for i, r := range head {
		title := ""
		snippet := ""
		if r.NoteView != nil {
			title = r.NoteView.Title
			// Keep the passage near the cross-encoder's input window (~512 tokens).
			// Longer passages measured strictly worse (see docs/dev/search_refactoring.md).
			snippet = generateSnippet(r.NoteView, 512)
		}
		docs[i] = title + "\n" + snippet
	}

	order, err := reranker.New(cfg.BaseURL, cfg.Model).Rerank(ctx, query, docs)
	if err != nil || len(order) == 0 {
		env.Logger().Warn("rerank failed", "error", err)
		return results
	}

	reordered := make([]appmodel.SearchResult, 0, len(results))
	seen := make([]bool, len(head))
	for _, idx := range order {
		if idx >= 0 && idx < len(head) && !seen[idx] {
			seen[idx] = true
			reordered = append(reordered, head[idx])
		}
	}
	// Keep any head items the reranker dropped, in their original order.
	for i, r := range head {
		if !seen[i] {
			reordered = append(reordered, r)
		}
	}
	// Preserve the tail beyond TopN unchanged.
	reordered = append(reordered, results[n:]...)

	if cfg.OutputK > 0 && len(reordered) > cfg.OutputK {
		reordered = reordered[:cfg.OutputK]
	}
	return reordered
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
			// Note exists in text results too — accumulate score, mark as hybrid
			e.rrfScore += score
			e.result.MatchOrigin = appmodel.SearchMatchHybrid
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

	if len(finalResults) > 20 {
		finalResults = finalResults[:20]
	}

	return finalResults
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
	text = trimWhitespace(text)
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

func trimWhitespace(s string) string {
	// Simple trim of leading/trailing whitespace and normalize internal whitespace
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
// which is guaranteed by the embedding server (embedding-server/server.py
// normalize_embeddings=True). Using dot product avoids the redundant sqrt
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
