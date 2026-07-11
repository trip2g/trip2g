package reranker

import (
	"context"
	"math"
	"sort"
	"time"

	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// BlendRRF applies an optional cross-encoder rerank to a fused candidate set
// and BLENDS the cross-encoder score with the first-stage RRF score rather
// than replacing the order. The blend keeps RRF as a strong prior:
//
//	final = (1-w)*rrf_norm + w*ce_norm
//
// where rrf_norm and ce_norm are min-max normalised to [0,1] over the reranked
// head, and w = BlendWeight. Only candidates with a window-sized passage (from
// the vector lane) are rescored — text-only results with no passage keep their
// stage-1 RRF score and are never truncated into the CE window. When the
// reranker is disabled it is a pure no-op; on any error it returns the input
// unchanged (graceful degradation). See docs/dev/reranker.md.
//
// Shared by the site search (internal/case/sitesearch) and the MCP search
// (internal/case/mcp) so both lanes rerank identically.
func BlendRRF(
	ctx context.Context,
	cfg features.RerankerConfig,
	log logger.Logger,
	query string,
	results []model.SearchResult,
	passageByURL map[string]string,
) []model.SearchResult {
	if !cfg.Enabled || len(results) < 2 {
		return results
	}

	n := cfg.TopN
	if n > len(results) {
		n = len(results)
	}
	head := results[:n]

	docs, docIdx := buildRerankDocs(head, passageByURL)
	if len(docs) < 2 {
		return results // nothing meaningful to reorder
	}

	client := NewWithTimeout(cfg.BaseURL, cfg.Model, time.Duration(cfg.TimeoutSeconds)*time.Second)
	scored, err := client.Rerank(ctx, query, docs)
	if err != nil || len(scored) == 0 {
		log.Warn("rerank failed", "error", err)
		return results
	}

	ceScore, ceMin, ceMax := collectCEScores(scored, docIdx)

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
	blended := make([]model.SearchResult, len(head))
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

	out := make([]model.SearchResult, 0, len(results))
	out = append(out, blended...)
	out = append(out, results[n:]...) // tail beyond TopN unchanged
	return out
}

// buildRerankDocs builds cross-encoder documents ("title\npassage") only for
// head candidates that carry a window-sized passage. The returned docIdx maps a
// reranker doc position back to its index in head.
func buildRerankDocs(head []model.SearchResult, passageByURL map[string]string) ([]string, []int) {
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
	return docs, docIdx
}

// collectCEScores maps cross-encoder scores back onto head positions via docIdx
// and returns the score-by-head-index map plus the min/max CE score range.
func collectCEScores(scored []Result, docIdx []int) (map[int]float64, float64, float64) {
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
	return ceScore, ceMin, ceMax
}

// normalize maps v into [0,1] by min-max over [lo,hi]. A degenerate range
// (hi==lo) returns 0.5 so a tied first stage contributes neutrally.
func normalize(v, lo, hi float64) float64 {
	if hi <= lo {
		return 0.5
	}
	return (v - lo) / (hi - lo)
}
