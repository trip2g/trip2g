package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"trip2g/internal/features"
	"trip2g/internal/model"
	"trip2g/internal/openai"
)

type Env interface {
	LatestNoteViews() *model.NoteViews
	SearchLatestNotes(query string) ([]model.SearchResult, error)
	Features() features.Features
	OpenAI() *openai.Client
}

func Resolve(ctx context.Context, env Env, req Request) Response {
	switch req.Method {
	case "initialize":
		return handleInitialize(env, req.ID)
	case "notifications/initialized":
		// Client notification, no response needed
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return handleToolsList(env, req.ID)
	case "tools/call":
		return handleToolsCall(ctx, env, req)
	default:
		return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: "+req.Method)
	}
}

func handleInitialize(env Env, id any) Response {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "trip2g-mcp",
			"version": "1.0.0",
		},
	}

	// Look for note with mcp_method: initialize
	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod == "initialize" {
			content := string(note.Content)
			content = stripFrontmatter(content)
			result["instructions"] = content
			break
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func handleToolsList(env Env, id any) Response {
	tools := []Tool{
		{
			Name:        "search",
			Description: "Search notes by query (hybrid: text + semantic)",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "similar",
			Description: "Find similar notes by text (vector similarity)",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"text":  {Type: "string", Description: "Text to find similar notes for"},
					"limit": {Type: "number", Description: "Max number of results (default 10)"},
				},
				Required: []string{"text"},
			},
		},
	}

	// Add dynamic methods from notes with mcp_method
	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod != "" && note.MCPMethod != "initialize" {
			tools = append(tools, Tool{
				Name:        note.MCPMethod,
				Description: note.MCPDescription,
				InputSchema: &InputSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			})
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  ListToolsResult{Tools: tools},
	}
}

func handleToolsCall(ctx context.Context, env Env, req Request) Response {
	var params CallToolParams
	err := json.Unmarshal(req.Params, &params)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params: "+err.Error())
	}

	switch params.Name {
	case "search":
		return handleSearch(ctx, env, req.ID, params.Arguments)
	case "similar":
		return handleSimilar(ctx, env, req.ID, params.Arguments)
	default:
		return handleDynamicMethod(env, req.ID, params.Name)
	}
}

func handleSearch(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	var args SearchArguments
	err := json.Unmarshal(argsRaw, &args)
	if err != nil {
		return errorResponse(id, ErrCodeInvalidParams, "Invalid search arguments: "+err.Error())
	}

	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}

	// Text search
	results, err := env.SearchLatestNotes(args.Query)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Search failed: "+err.Error())
	}

	// Add vector search results if enabled
	if env.Features().VectorSearch.Enabled && env.OpenAI() != nil {
		vectorResults, vecErr := vectorSearch(ctx, env, args.Query, 10)
		if vecErr == nil {
			results = mergeResults(results, vectorResults)
		}
	}

	// Format response
	var sb strings.Builder
	if len(results) == 0 {
		sb.WriteString("No results found for: " + args.Query)
	} else {
		sb.WriteString(fmt.Sprintf("Found %d notes:\n\n", len(results)))
		for i, r := range results {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("\n... and %d more", len(results)-10))
				break
			}
			title := r.NoteView.Title
			if r.HighlightedTitle != nil {
				title = *r.HighlightedTitle
			}
			sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, title, r.URL))
			if len(r.HighlightedContent) > 0 {
				sb.WriteString(fmt.Sprintf("   %s\n", r.HighlightedContent[0]))
			}
			sb.WriteString("\n")
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: CallToolResult{
			Content: []Content{{Type: "text", Text: sb.String()}},
		},
	}
}

func handleSimilar(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	var args SimilarArguments
	err := json.Unmarshal(argsRaw, &args)
	if err != nil {
		return errorResponse(id, ErrCodeInvalidParams, "Invalid similar arguments: "+err.Error())
	}

	if args.Text == "" {
		return errorResponse(id, ErrCodeInvalidParams, "text is required")
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 10
	}

	if env.OpenAI() == nil {
		return errorResponse(id, ErrCodeInternal, "Vector search not configured")
	}

	results, err := vectorSearch(ctx, env, args.Text, limit)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Similar search failed: "+err.Error())
	}

	// Format response
	var sb strings.Builder
	if len(results) == 0 {
		sb.WriteString("No similar notes found")
	} else {
		sb.WriteString(fmt.Sprintf("Found %d similar notes:\n\n", len(results)))
		for i, r := range results {
			sb.WriteString(fmt.Sprintf("%d. %s (%.2f)\n   %s\n\n", i+1, r.NoteView.Title, r.Score, r.URL))
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: CallToolResult{
			Content: []Content{{Type: "text", Text: sb.String()}},
		},
	}
}

func handleDynamicMethod(env Env, id any, methodName string) Response {
	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod == methodName {
			// Return note content without frontmatter
			content := string(note.Content)
			content = stripFrontmatter(content)

			return Response{
				JSONRPC: "2.0",
				ID:      id,
				Result: CallToolResult{
					Content: []Content{{Type: "text", Text: content}},
				},
			}
		}
	}

	return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+methodName)
}

func stripFrontmatter(content string) string {
	if len(content) < 3 || content[:3] != "---" {
		return content
	}

	// Find closing ---
	for i := 4; i < len(content)-2; i++ {
		if content[i] == '-' && content[i+1] == '-' && content[i+2] == '-' {
			result := content[i+3:]
			// Trim leading newlines
			for len(result) > 0 && (result[0] == '\n' || result[0] == '\r') {
				result = result[1:]
			}
			return result
		}
	}

	return content
}

func vectorSearch(ctx context.Context, env Env, query string, limit int) ([]model.SearchResult, error) {
	embedding, err := env.OpenAI().CreateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	noteViews := env.LatestNoteViews()

	type scored struct {
		note  *model.NoteView
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

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if len(scores) > limit {
		scores = scores[:limit]
	}

	results := make([]model.SearchResult, 0, len(scores))
	for _, s := range scores {
		results = append(results, model.SearchResult{
			NoteView: s.note,
			URL:      s.note.Permalink,
			Score:    s.score,
		})
	}

	return results, nil
}

func mergeResults(textResults, vectorResults []model.SearchResult) []model.SearchResult {
	if len(vectorResults) == 0 {
		return textResults
	}

	const textWeight = 0.6
	const vectorWeight = 0.4

	maxTextScore := 0.0
	for _, r := range textResults {
		if r.Score > maxTextScore {
			maxTextScore = r.Score
		}
	}

	type merged struct {
		result      model.SearchResult
		textScore   float64
		vectorScore float64
	}

	resultMap := make(map[string]*merged)

	for _, r := range textResults {
		normalizedScore := 0.0
		if maxTextScore > 0 {
			normalizedScore = r.Score / maxTextScore
		}
		resultMap[r.URL] = &merged{
			result:    r,
			textScore: normalizedScore,
		}
	}

	for _, r := range vectorResults {
		if existing, ok := resultMap[r.URL]; ok {
			existing.vectorScore = r.Score
		} else {
			title := r.NoteView.Title
			r.HighlightedTitle = &title
			resultMap[r.URL] = &merged{
				result:      r,
				vectorScore: r.Score,
			}
		}
	}

	var finalResults []model.SearchResult
	for _, m := range resultMap {
		m.result.Score = m.textScore*textWeight + m.vectorScore*vectorWeight
		finalResults = append(finalResults, m.result)
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Score > finalResults[j].Score
	})

	if len(finalResults) > 20 {
		finalResults = finalResults[:20]
	}

	return finalResults
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
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

func errorResponse(id any, code int, message string) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}
