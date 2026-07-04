package sitesearch

import (
	"context"
	"fmt"
	"math"
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

//nolint:gocognit // multi-source search merge with per-result auth, scoping, and RRF ranking
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
	// passageByURL holds the best-matching chunk passage per note (window-sized),
	// used by the optional reranker to avoid feeding truncated whole notes.
	var passageByURL map[string]string
	if env.Features().VectorSearch.Enabled && env.OpenAI() != nil {
		vectorResults, passages, vectorErr := vectorSearch(ctx, env, input.Query, useLatest)
		if vectorErr != nil {
			// Log error but don't fail - text search still works
			env.Logger().Warn("vector search failed", "error", vectorErr)
		} else {
			passageByURL = passages
			results = mergeResults(results, vectorResults)
		}
	}

	// Optional second-stage cross-encoder rerank, BLENDED with the RRF order.
	results = rerankResults(ctx, env, input.Query, results, passageByURL)

	// Filter results based on permissions
	conn := model.SearchConnection{}
	hiddenResults := []appmodel.SearchResult{}

	for _, res := range results {
		if res.NoteView != nil { //nolint:nestif // per-result auth checks require nil-guard, scope check, and read-pattern gate
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

func vectorSearch(ctx context.Context, env Env, query string, useLatest bool) ([]appmodel.SearchResult, map[string]string, error) {
	queryPrefix := env.Features().VectorSearch.ResolvedQueryPrefix()
	embedding, err := env.OpenAI().CreateEmbedding(ctx, queryPrefix+query)
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
	// returns L2-normalised unit vectors (embedding-server/server.py normalize_embeddings=True),
	// making cosine ≡ dot product at lower compute cost.
	scanStart := time.Now()
	var candidates []scored
	for _, c := range chunks {
		sim := dotSimilarity(embedding.Vector, c.Embedding)
		candidates = append(candidates, scored{c.NotePath, c, sim})
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
		results = append(results, appmodel.SearchResult{
			NoteView:           note,
			URL:                note.Permalink,
			Score:              c.sim,
			MatchOrigin:        appmodel.SearchMatchVector,
			HighlightedTitle:   &title,
			HighlightedContent: []string{snippetFromChunk(c.chunk.Content, 200)},
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
	return trimWhitespace(content)
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

// rerankResults applies an optional cross-encoder rerank to the fused candidate
// set and BLENDS the cross-encoder score with the first-stage RRF score rather
// than replacing the order. The blend keeps RRF as a strong prior:
//
//	final = (1-w)*rrf_norm + w*ce_norm
//
// where rrf_norm and ce_norm are min-max normalised to [0,1] over the reranked
// head, and w = BlendWeight. Only candidates with a window-sized passage (from
// the vector lane) are rescored — text-only results with no passage keep their
// stage-1 RRF score and are never truncated into the CE window. On any error it
// returns the input unchanged (graceful degradation). See docs/dev/reranker.md.
func rerankResults(
	ctx context.Context,
	env Env,
	query string,
	results []appmodel.SearchResult,
	passageByURL map[string]string,
) []appmodel.SearchResult {
	cfg := env.Features().VectorSearch.Reranker
	if !cfg.Enabled || len(results) < 2 {
		return results
	}

	n := cfg.TopN
	if n > len(results) {
		n = len(results)
	}
	head := results[:n]

	// Build CE documents only for candidates that have a window-sized passage.
	// docIdx maps a reranker doc position back to its index in head.
	var docs []string
	var docIdx []int
	for i, r := range head {
		passage, ok := passageByURL[r.URL]
		if !ok || passage == "" {
			continue
		}
		title := ""
		if r.NoteView != nil {
			title = r.NoteView.Title
		}
		docs = append(docs, title+"\n"+passage)
		docIdx = append(docIdx, i)
	}
	if len(docs) < 2 {
		return results // nothing meaningful to reorder
	}

	client := reranker.NewWithTimeout(cfg.BaseURL, cfg.Model, time.Duration(cfg.TimeoutSeconds)*time.Second)
	scored, err := client.Rerank(ctx, query, docs)
	if err != nil || len(scored) == 0 {
		env.Logger().Warn("rerank failed", "error", err)
		return results
	}

	// Collect CE scores back onto head positions.
	ceScore := make(map[int]float64, len(scored))
	ceMin, ceMax := math.Inf(1), math.Inf(-1)
	for _, s := range scored {
		if s.Index < 0 || s.Index >= len(docIdx) {
			continue
		}
		h := docIdx[s.Index]
		ceScore[h] = s.Score
		if s.Score < ceMin {
			ceMin = s.Score
		}
		if s.Score > ceMax {
			ceMax = s.Score
		}
	}

	// Min-max range of the first-stage RRF scores over the head.
	rrfMin, rrfMax := math.Inf(1), math.Inf(-1)
	for _, r := range head {
		if r.Score < rrfMin {
			rrfMin = r.Score
		}
		if r.Score > rrfMax {
			rrfMax = r.Score
		}
	}

	w := cfg.BlendWeight
	blended := make([]appmodel.SearchResult, len(head))
	copy(blended, head)
	for i := range blended {
		rrfNorm := normalize(blended[i].Score, rrfMin, rrfMax)
		ce, ok := ceScore[i]
		if !ok {
			// No CE score for this candidate (no passage): keep its RRF prior,
			// blended against the neutral CE midpoint so it isn't unfairly sunk.
			blended[i].Score = (1-w)*rrfNorm + w*0.5
			continue
		}
		ceNorm := normalize(ce, ceMin, ceMax)
		blended[i].Score = (1-w)*rrfNorm + w*ceNorm
	}

	sort.SliceStable(blended, func(i, j int) bool {
		if blended[i].Score != blended[j].Score {
			return blended[i].Score > blended[j].Score
		}
		return blended[i].URL < blended[j].URL
	})

	out := make([]appmodel.SearchResult, 0, len(results))
	out = append(out, blended...)
	out = append(out, results[n:]...) // tail beyond TopN unchanged

	if cfg.OutputK > 0 && len(out) > cfg.OutputK {
		out = out[:cfg.OutputK]
	}
	return out
}

// normalize maps v into [0,1] by min-max over [lo,hi]. A degenerate range
// (hi==lo) returns 0.5 so a tied first stage contributes neutrally.
func normalize(v, lo, hi float64) float64 {
	if hi <= lo {
		return 0.5
	}
	return (v - lo) / (hi - lo)
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
