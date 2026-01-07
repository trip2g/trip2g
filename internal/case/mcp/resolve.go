package mcp

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg mcp_test . Env

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"trip2g/internal/case/similarnotes"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/openai"
	"trip2g/internal/ptr"
)

const (
	// Search and display limits.
	DefaultVectorSearchLimit = 10
	DefaultDisplayLimit      = 10
	DefaultSimilarLimit      = 10
	MaxSimilarLimit          = 100
	MaxMergedResults         = 20

	// Hybrid search weights.
	TextSearchWeight   = 0.6
	VectorSearchWeight = 0.4

	// MCP method names.
	MCPMethodInitialize = "initialize"
)

type Env interface {
	similarnotes.Env
	SearchLatestNotes(query string) ([]model.SearchResult, error)
	OpenAI() *openai.Client
	PublicURL() string
	Logger() logger.Logger
}

// unmarshalArgs unmarshals JSON arguments into the target type.
// Returns error response if unmarshaling fails.
func unmarshalArgs[T any](argsRaw json.RawMessage, id any, toolName string) (*T, *Response) {
	var args T
	err := json.Unmarshal(argsRaw, &args)
	if err != nil {
		resp := errorResponse(id, ErrCodeInvalidParams, fmt.Sprintf("Invalid %s arguments: %v", toolName, err))
		return nil, &resp
	}
	return &args, nil
}

// successResponse creates a successful JSON-RPC response.
func successResponse(id any, result any) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// textToolResult creates a CallToolResult with text content.
func textToolResult(text string) CallToolResult {
	return CallToolResult{
		Content: []Content{{Type: "text", Text: text}},
	}
}

func Resolve(ctx context.Context, env Env, req Request) Response {
	switch req.Method {
	case MCPMethodInitialize:
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
		if note.MCPMethod == MCPMethodInitialize {
			content := string(note.Content)
			content = stripFrontmatter(content)
			result["instructions"] = content
			break
		}
	}

	return successResponse(id, result)
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
			Description: "Find similar notes by path (vector similarity)",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":  {Type: "string", Description: "Note path from search results"},
					"limit": {Type: "number", Description: "Max number of results (default 10)"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "note_html",
			Description: "Get HTML content of a note by path",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Note path from search results"},
				},
				Required: []string{"path"},
			},
		},
	}

	// Add dynamic methods from notes with mcp_method
	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod != "" && note.MCPMethod != MCPMethodInitialize {
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

	return successResponse(id, ListToolsResult{Tools: tools})
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
	case "note_html":
		return handleNoteHTML(env, req.ID, params.Arguments)
	default:
		return handleDynamicMethod(env, req.ID, params.Name)
	}
}

func handleSearch(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleSearch")

	args, errResp := unmarshalArgs[SearchArguments](argsRaw, id, "search")
	if errResp != nil {
		return *errResp
	}

	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}

	// Text search
	results, err := env.SearchLatestNotes(args.Query)
	if err != nil {
		log.Error("text search failed", "error", err, "query", args.Query)
		return errorResponse(id, ErrCodeInternal, "Search failed: "+err.Error())
	}

	// Add vector search results if enabled
	if env.Features().VectorSearch.Enabled && env.OpenAI() != nil {
		vectorResults, vecErr := vectorSearch(ctx, env, args.Query, DefaultVectorSearchLimit)
		if vecErr == nil {
			results = mergeResults(results, vectorResults)
		} else {
			log.Warn("vector search failed", "error", vecErr, "query", args.Query)
		}
	}

	// Format response
	var sb strings.Builder
	if len(results) == 0 {
		sb.WriteString("No results found for: " + args.Query)
	} else {
		sb.WriteString(fmt.Sprintf("Found %d notes:\n\n", len(results)))
		for i, r := range results {
			if i >= DefaultDisplayLimit {
				sb.WriteString(fmt.Sprintf("\n... and %d more", len(results)-DefaultDisplayLimit))
				break
			}
			title := r.NoteView.Title
			if r.HighlightedTitle != nil {
				title = *r.HighlightedTitle
			}
			sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n", i+1, title, r.NoteView.Path, env.PublicURL()+r.NoteView.Permalink))
			if len(r.HighlightedContent) > 0 {
				sb.WriteString(fmt.Sprintf("   %s\n", r.HighlightedContent[0]))
			}
			sb.WriteString("\n")
		}
	}

	log.Debug("search completed", "query", args.Query, "results", len(results))

	return successResponse(id, textToolResult(sb.String()))
}

func handleSimilar(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleSimilar")

	args, errResp := unmarshalArgs[SimilarArguments](argsRaw, id, "similar")
	if errResp != nil {
		return *errResp
	}

	if args.Path == "" {
		return errorResponse(id, ErrCodeInvalidParams, "path is required")
	}

	// Validate and normalize limit
	limit := args.Limit
	if limit <= 0 {
		limit = DefaultSimilarLimit
	} else if limit > MaxSimilarLimit {
		log.Warn("limit exceeds maximum, capping", "requested", limit, "max", MaxSimilarLimit)
		limit = MaxSimilarLimit
	}

	input := graphmodel.SimilarNotesInput{
		Path:  args.Path,
		Limit: ptr.To(int32(limit)),
	}

	results, err := similarnotes.Resolve(ctx, env, input)
	if err != nil {
		log.Error("similar search failed", "error", err, "path", args.Path)
		return errorResponse(id, ErrCodeInternal, "Similar search failed: "+err.Error())
	}

	// Format response
	var sb strings.Builder
	if len(results) == 0 {
		sb.WriteString("No similar notes found")
	} else {
		sb.WriteString(fmt.Sprintf("Found %d similar notes:\n\n", len(results)))
		for i, r := range results {
			sb.WriteString(fmt.Sprintf("%d. %s (%.2f)\n   %s\n   %s\n\n", i+1, r.Note.Title, r.Score, r.Note.Path, env.PublicURL()+r.Note.Path))
		}
	}

	log.Debug("similar search completed", "path", args.Path, "results", len(results))

	return successResponse(id, textToolResult(sb.String()))
}

func handleNoteHTML(env Env, id any, argsRaw json.RawMessage) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleNoteHTML")

	args, errResp := unmarshalArgs[NoteHTMLArguments](argsRaw, id, "note_html")
	if errResp != nil {
		return *errResp
	}

	if args.Path == "" {
		return errorResponse(id, ErrCodeInvalidParams, "path is required")
	}

	noteViews := env.LatestNoteViews()
	note := noteViews.PathMap[args.Path]
	if note == nil {
		log.Warn("note not found", "path", args.Path)
		return errorResponse(id, ErrCodeInvalidParams, "Note not found: "+args.Path)
	}

	log.Debug("note html retrieved", "path", args.Path)

	return successResponse(id, textToolResult(string(note.HTML)))
}

func handleDynamicMethod(env Env, id any, methodName string) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleDynamicMethod")

	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod == methodName {
			content := string(note.Content)
			content = stripFrontmatter(content)

			log.Debug("dynamic method executed", "method", methodName, "note_path", note.Path)

			return successResponse(id, textToolResult(content))
		}
	}

	log.Warn("dynamic method not found", "method", methodName)
	return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+methodName)
}

func stripFrontmatter(content string) string {
	// Check for frontmatter start (support both \n and \r\n)
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return content
	}

	// Determine line ending and skip opening "---"
	start := 4 // "---\n"
	if strings.HasPrefix(content, "---\r\n") {
		start = 5 // "---\r\n"
	}

	if len(content) <= start {
		return content
	}

	// Find closing "---" at the start of a line
	remaining := content[start:]
	idx := strings.Index(remaining, "\n---")
	if idx == -1 {
		idx = strings.Index(remaining, "\r\n---")
		if idx == -1 {
			return content
		}
		// Skip past "\r\n---"
		result := remaining[idx+5:]
		// Check if there's a newline after closing ---
		if strings.HasPrefix(result, "\n") {
			result = result[1:]
		} else if strings.HasPrefix(result, "\r\n") {
			result = result[2:]
		}
		return strings.TrimLeft(result, "\r\n")
	}

	// Skip past "\n---"
	result := remaining[idx+4:]
	// Check if there's a newline after closing ---
	if strings.HasPrefix(result, "\n") {
		result = result[1:]
	} else if strings.HasPrefix(result, "\r\n") {
		result = result[2:]
	}

	return strings.TrimLeft(result, "\r\n")
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
		m.result.Score = m.textScore*TextSearchWeight + m.vectorScore*VectorSearchWeight
		finalResults = append(finalResults, m.result)
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Score > finalResults[j].Score
	})

	if len(finalResults) > MaxMergedResults {
		finalResults = finalResults[:MaxMergedResults]
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
