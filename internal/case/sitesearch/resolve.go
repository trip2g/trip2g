package sitesearch

import (
	"context"
	"fmt"
	"math"
	"sort"

	"trip2g/internal/features"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/openai"
	"trip2g/internal/usertoken"

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

	// Filter results based on permissions
	conn := model.SearchConnection{}
	hiddenResults := []appmodel.SearchResult{}

	for _, res := range results {
		if res.NoteView != nil {
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

func vectorSearch(ctx context.Context, env Env, query string, useLatest bool) ([]appmodel.SearchResult, error) {
	// Generate query embedding
	embedding, err := env.OpenAI().CreateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Get note views
	var noteViews *appmodel.NoteViews
	if useLatest {
		noteViews = env.LatestNoteViews()
	} else {
		noteViews = env.LiveNoteViews()
	}

	// Calculate similarity for all notes with embeddings
	type scored struct {
		note  *appmodel.NoteView
		score float64
	}

	var scores []scored
	for _, note := range noteViews.List {
		if len(note.Embedding) == 0 {
			continue
		}

		similarity := cosineSimilarity(embedding.Vector, note.Embedding)
		scores = append(scores, scored{note: note, score: similarity})
	}

	// Sort by similarity (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Take top 30 results (more candidates improve RRF quality)
	limit := 30
	if len(scores) < limit {
		limit = len(scores)
	}

	results := make([]appmodel.SearchResult, 0, limit)
	for i := range limit {
		s := scores[i]
		results = append(results, appmodel.SearchResult{
			NoteView: s.note,
			URL:      s.note.Permalink,
			Score:    s.score,
		})
	}

	return results, nil
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
			// Note exists in text results too — accumulate score, keep text highlights
			e.rrfScore += score
		} else {
			// Vector-only result — generate snippet for display
			title := r.NoteView.Title
			r.HighlightedTitle = &title
			r.HighlightedContent = []string{generateSnippet(r.NoteView, 150)}
			resultMap[r.URL] = &entry{result: r, rrfScore: score}
		}
	}

	finalResults := make([]appmodel.SearchResult, 0, len(resultMap))
	for _, e := range resultMap {
		e.result.Score = e.rrfScore
		finalResults = append(finalResults, e.result)
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Score > finalResults[j].Score
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
	if len(text) > maxLen {
		// Try to break at word boundary
		text = text[:maxLen]
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

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
