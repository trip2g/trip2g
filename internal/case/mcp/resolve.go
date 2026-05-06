package mcp

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg mcp_test . Env

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/similarnotes"
	"trip2g/internal/db"
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

	// Hybrid search rank constant.
	rrfK = 60

	// MCP method names.
	MCPMethodInitialize = "initialize"
)

type Env interface {
	similarnotes.Env
	model.FederationClientFactory
	SearchLatestNotes(query string) ([]model.SearchResult, error)
	LatestNoteChunks() []model.NoteChunk
	OpenAI() *openai.Client
	PublicURL() string
	NoteURL(note *model.NoteView) string
	Logger() logger.Logger
	FederationSecretByKBURL(ctx context.Context, kbURL string) (db.FederationSecret, bool, error)
	FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error)
	ListFederationSecretSubgraphsByKID(ctx context.Context, kid string) ([]string, error)
	DecryptData([]byte) ([]byte, error)
	FederationMaxDepth() int
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

func structuredToolResult(text string, structured any) CallToolResult {
	return CallToolResult{
		Content:           []Content{{Type: "text", Text: text}},
		StructuredContent: structured,
	}
}

func Resolve(ctx context.Context, env Env, req Request) Response {
	switch req.Method {
	case MCPMethodInitialize:
		return handleInitialize(ctx, env, req.ID, req.MethodOverride)
	case "notifications/initialized":
		// Client notification, no response needed
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return handleToolsList(ctx, env, req.ID)
	case "tools/call":
		return handleToolsCall(ctx, env, req)
	default:
		return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: "+req.Method)
	}
}

func handleInitialize(ctx context.Context, env Env, id any, methodOverride string) Response {
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

	target := methodOverride
	if target == "" {
		target = MCPMethodInitialize
	}

	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod != target {
			continue
		}
		ok, err := canReadMCPNote(ctx, env, note)
		if err != nil {
			return errorResponse(id, ErrCodeInternal, "Instructions access check failed: "+err.Error())
		}
		if !ok {
			// Note exists but user cannot read it.
			// For explicit ?method= overrides this is an error; for default initialize it is a silent skip.
			if methodOverride != "" {
				return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+target)
			}
			break
		}
		content := string(note.Content)
		content = stripFrontmatter(content)
		result["instructions"] = content
		break
	}

	// Explicit ?method= with no matching (or inaccessible) note is an error
	if methodOverride != "" {
		if _, hasInstructions := result["instructions"]; !hasInstructions {
			return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+methodOverride)
		}
	}

	return successResponse(id, result)
}

var reservedMCPTools = map[string]bool{ //nolint:gochecknoglobals // immutable set of built-in tool names
	"search":              true,
	"similar":             true,
	"note_html":           true,
	"federated_search":    true,
	"federated_similar":   true,
	"federated_note_html": true,
	MCPMethodInitialize:   true,
}

func handleToolsList(ctx context.Context, env Env, id any) Response {
	tools := []Tool{
		{
			Name:        "search",
			Description: "Search notes by query and return note ids, snippets, and match ids. After search, open the best result with note_html(pid=..., match_id=...) when a match id is available.",
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
			Description: "Find related notes from a known note id, href, or path. Use this after opening a promising note when you need nearby context.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":    {Type: "string", Description: "Note path from search or note_html"},
					"href":    {Type: "string", Description: "Note href from search results"},
					"pid":     {Type: "number", Description: "Stable note id from search or HTML data-pid"},
					"note_id": {Type: "number", Description: "Stable note id from search results"},
					"limit":   {Type: "number", Description: "Max number of results (default 10)"},
				},
			},
		},
		{
			Name:        "note_html",
			Description: "Read a note by pid, note_id, href, or path. Pass match_id for a focused chunk window. Pass toc_path to read a specific section (use the path array from search result TOC).",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":          {Type: "string", Description: "Note path from search results"},
					"href":          {Type: "string", Description: "Note href from search results"},
					"pid":           {Type: "number", Description: "Stable note id from search results or HTML data-pid"},
					"note_id":       {Type: "number", Description: "Stable note id from search results"},
					"match_id":      {Type: "string", Description: "Focused match id from search results, such as p32:c4"},
					"context_words": {Type: "number", Description: "Optional future hint for expanding focused reads"},
					"toc_path": {
						Type:        "array",
						Description: "Breadcrumb path to a specific section, e.g. [\"Chapter 1\", \"Introduction\"]. Use path from search result toc items.",
						Items:       &Property{Type: "string"},
					},
				},
			},
		},
		{
			Name:        "federated_search",
			Description: "Search connected knowledge bases. Pass kb_id for one base, kb_ids for selected bases, or omit both to fan out.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":  {Type: "string", Description: "Search query"},
					"kb_id":  {Type: "string", Description: "Target knowledge base id"},
					"kb_ids": {Type: "array", Description: "Target knowledge base ids", Items: &Property{Type: "string"}},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "federated_similar",
			Description: "Find remote notes similar to a known note reference inside a connected knowledge base.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"kb_id":   {Type: "string", Description: "Target knowledge base id"},
					"path":    {Type: "string", Description: "Remote note path"},
					"href":    {Type: "string", Description: "Remote note href"},
					"pid":     {Type: "number", Description: "Remote stable note id"},
					"note_id": {Type: "number", Description: "Remote stable note id"},
					"limit":   {Type: "number", Description: "Max number of results"},
				},
				Required: []string{"kb_id"},
			},
		},
		{
			Name:        "federated_note_html",
			Description: "Read a remote note by pid, note_id, href, path, or match_id inside a connected knowledge base.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"kb_id":    {Type: "string", Description: "Target knowledge base id"},
					"path":     {Type: "string", Description: "Remote note path"},
					"href":     {Type: "string", Description: "Remote note href"},
					"pid":      {Type: "number", Description: "Remote stable note id"},
					"note_id":  {Type: "number", Description: "Remote stable note id"},
					"match_id": {Type: "string", Description: "Focused match id from remote search results"},
				},
				Required: []string{"kb_id"},
			},
		},
	}

	// Append dynamic tools from notes with mcp_method (excluding reserved names)
	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod == "" || reservedMCPTools[note.MCPMethod] {
			continue
		}
		ok, err := canReadMCPNote(ctx, env, note)
		if err != nil || !ok {
			continue
		}
		desc := note.MCPDescription
		if desc == "" {
			desc = note.Title
		}
		tools = append(tools, Tool{
			Name:        note.MCPMethod,
			Description: desc,
			InputSchema: &InputSchema{Type: "object", Properties: map[string]Property{}},
		})
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
		return handleNoteHTML(ctx, env, req.ID, params.Arguments)
	case "federated_search":
		return handleFederatedSearch(ctx, env, req.ID, params.Arguments)
	case "federated_similar":
		return handleFederatedSimilar(ctx, env, req.ID, params.Arguments)
	case "federated_note_html":
		return handleFederatedNoteHTML(ctx, env, req.ID, params.Arguments)
	default:
		return handleDynamicMethod(ctx, env, req.ID, params.Name)
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
	results, err = filterSearchResults(ctx, env, results)
	if err != nil {
		log.Error("search access check failed", "error", err, "query", args.Query)
		return errorResponse(id, ErrCodeInternal, "Search failed: "+err.Error())
	}

	payload := buildSearchPayload(args.Query, results, env.NoteURL, env.LatestNoteChunks())

	// Format response
	var sb strings.Builder
	if len(payload.Results) == 0 {
		sb.WriteString("No results found for: " + args.Query)
	} else {
		sb.WriteString(fmt.Sprintf("Found %d notes:\n\n", len(payload.Results)))
		for i, r := range payload.Results {
			if i >= DefaultDisplayLimit {
				sb.WriteString(fmt.Sprintf("\n... and %d more", len(payload.Results)-DefaultDisplayLimit))
				break
			}
			sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n", i+1, r.Title, r.NotePath, r.URL))
			if len(r.Matches) > 0 {
				sb.WriteString(fmt.Sprintf("   %s\n", r.Matches[0].Snippet))
				sb.WriteString(fmt.Sprintf("   match_id: %s\n", r.Matches[0].MatchID))
			}
			sb.WriteString("\n")
		}
	}

	log.Debug("search completed", "query", args.Query, "results", len(results))

	return successResponse(id, structuredToolResult(sb.String(), payload))
}

func filterSearchResults(ctx context.Context, env Env, results []model.SearchResult) ([]model.SearchResult, error) {
	filtered := make([]model.SearchResult, 0, len(results))
	for _, r := range results {
		if r.NoteView == nil {
			continue
		}
		if r.NoteView.IsSystem() || r.NoteView.ExcludeSearch {
			continue
		}

		ok, err := canReadMCPNote(ctx, env, r.NoteView)
		if err != nil {
			return nil, fmt.Errorf("check note access: %w", err)
		}
		if ok {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

type federationAuthContextKey struct{}

type federationAuthContext struct {
	KID              string
	AllowedSubgraphs []string
}

func contextWithFederationAuth(ctx context.Context, kid string, allowedSubgraphs []string) context.Context {
	return context.WithValue(ctx, federationAuthContextKey{}, federationAuthContext{
		KID:              kid,
		AllowedSubgraphs: append([]string(nil), allowedSubgraphs...),
	})
}

func federationAuthFromContext(ctx context.Context) (federationAuthContext, bool) {
	auth, ok := ctx.Value(federationAuthContextKey{}).(federationAuthContext)
	return auth, ok
}

type mcpAPIKeyAuthContextKey struct{}

type mcpAPIKeyAuthInfo struct {
	adminTools bool
}

func contextWithMCPAPIKeyAuth(ctx context.Context, adminTools bool) context.Context {
	return context.WithValue(ctx, mcpAPIKeyAuthContextKey{}, mcpAPIKeyAuthInfo{adminTools: adminTools})
}

func mcpAPIKeyAuthed(ctx context.Context) bool {
	_, ok := ctx.Value(mcpAPIKeyAuthContextKey{}).(mcpAPIKeyAuthInfo)
	return ok
}

func mcpAdminToolsEnabled(ctx context.Context) bool {
	info, ok := ctx.Value(mcpAPIKeyAuthContextKey{}).(mcpAPIKeyAuthInfo)
	return ok && info.adminTools
}

type federationDepthContextKey struct{}

func contextWithFederationDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, federationDepthContextKey{}, depth)
}

// FederationDepthFromContext returns the federation hop depth stored in ctx, or 0 if not set.
func FederationDepthFromContext(ctx context.Context) int {
	depth, _ := ctx.Value(federationDepthContextKey{}).(int)
	return depth
}

func canReadMCPNote(ctx context.Context, env Env, note *model.NoteView) (bool, error) {
	if mcpAPIKeyAuthed(ctx) {
		return true, nil // API key = admin, sees all notes
	}
	if auth, ok := federationAuthFromContext(ctx); ok {
		return canreadnote.ResolveWithSubgraphs(ctx, env, note, auth.AllowedSubgraphs)
	}
	return env.CanReadNote(ctx, note)
}

func buildSearchPayload(query string, results []model.SearchResult, noteURL func(*model.NoteView) string, chunks []model.NoteChunk) SearchResultPayload {
	payload := SearchResultPayload{Query: query}
	for _, r := range results {
		if r.NoteView == nil {
			continue
		}

		item := searchResultItemFromNote(r.NoteView, r.Score, noteURL)
		for i, snippet := range r.HighlightedContent {
			matchID := fmt.Sprintf("p%d:m%d", r.NoteView.PathID, i+1)
			chunkIndex := 0
			if r.ChunkIndex != nil {
				chunkIndex = *r.ChunkIndex
			} else if nearest, ok := nearestChunkIndexForSnippet(r.NoteView, snippet, chunks); ok {
				chunkIndex = nearest
			}
			if chunkIndex > 0 || (r.ChunkIndex != nil && chunkIndex == 0) {
				matchID = fmt.Sprintf("p%d:c%d", r.NoteView.PathID, chunkIndex)
			}
			item.Matches = append(item.Matches, SearchMatch{
				MatchID:      matchID,
				ChunkIndex:   chunkIndex,
				Snippet:      snippet,
				ContextWords: 10,
				TOCPath:      tocPathForSnippet(string(r.NoteView.HTML), snippet),
			})
		}
		payload.Results = append(payload.Results, item)
	}
	return payload
}

func nearestChunkIndexForSnippet(note *model.NoteView, snippet string, chunks []model.NoteChunk) (int, bool) {
	normalizedSnippet := normalizeSearchSnippet(snippet)
	if normalizedSnippet == "" {
		return 0, false
	}

	bestIndex := -1
	bestScore := 0
	secondScore := 0

	for _, chunk := range chunks {
		if chunk.NotePath != note.Path {
			continue
		}
		normalizedChunk := normalizeSearchSnippet(snippetFromChunk(chunk.Content, 400))
		if normalizedChunk == "" {
			continue
		}

		var score int
		if strings.Contains(normalizedChunk, normalizedSnippet) || strings.Contains(normalizedSnippet, normalizedChunk) {
			score = 1000 + len(normalizedSnippet)
		} else {
			score = overlapTokenScore(normalizedSnippet, normalizedChunk)
		}

		if score > bestScore {
			secondScore = bestScore
			bestScore = score
			bestIndex = chunk.ChunkIndex
		} else if score > secondScore {
			secondScore = score
		}
	}

	if bestIndex < 0 {
		return 0, false
	}
	// Conservative gate: require a clear winner and non-trivial overlap.
	if bestScore < 3 || bestScore-secondScore < 2 {
		return 0, false
	}
	return bestIndex, true
}

func normalizeSearchSnippet(s string) string {
	replacer := strings.NewReplacer("<mark>", "", "</mark>", "")
	s = replacer.Replace(s)
	s = trimWhitespace(strings.ToLower(s))
	return s
}

func overlapTokenScore(a string, b string) int {
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	setB := make(map[string]struct{}, len(wordsB))
	for _, word := range wordsB {
		if len([]rune(word)) < 4 {
			continue
		}
		setB[word] = struct{}{}
	}

	score := 0
	for _, word := range wordsA {
		if len([]rune(word)) < 4 {
			continue
		}
		if _, ok := setB[word]; ok {
			score++
		}
	}
	return score
}

func searchResultItemFromNote(note *model.NoteView, score float64, noteURL func(*model.NoteView) string) SearchResultItem {
	item := SearchResultItem{
		Title:    note.Title,
		NoteID:   note.PathID,
		NotePath: note.Path,
		Href:     note.Permalink,
		URL:      noteURL(note),
		Kind:     noteKind(note),
		Score:    score,
		TOC:      buildNoteTOC(note.Headings),
	}
	if kb := model.NewMCPFederationNote(note); kb != nil {
		item.Kind = "federation_kb"
		item.TOC = nil // federation pointers have no local TOC
		item.Federation = &FederationRef{
			KBID:             kb.ID,
			KBURL:            kb.URL,
			AgentInstruction: federationAgentInstruction(kb.ID),
		}
	}
	return item
}

func federationAgentInstruction(kbID string) string {
	return fmt.Sprintf(
		`This is a knowledge base pointer. To search inside it, call federated_search with kb_id="%s". To open notes from it, call federated_note_html(note_id=..., kb_id="%s").`,
		kbID,
		kbID,
	)
}

func noteKind(note *model.NoteView) string {
	path := strings.ToLower(note.Path)
	if strings.Contains(note.Path, "Книги/") || strings.Contains(path, "meditations") {
		return "source"
	}
	if note.IsIndex {
		return "index"
	}
	return "note"
}

func handleSimilar(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleSimilar")

	args, errResp := unmarshalArgs[SimilarArguments](argsRaw, id, "similar")
	if errResp != nil {
		return *errResp
	}

	if args.Path == "" && args.Href == "" && args.PID == 0 && args.NoteID == 0 {
		return errorResponse(id, ErrCodeInvalidParams, "one of pid, note_id, path, or href is required")
	}

	noteViews := env.LatestNoteViews()
	sourceNote := resolveSimilarReference(noteViews, *args)
	if sourceNote == nil {
		log.Warn("source note not found", "path", args.Path, "href", args.Href, "pid", args.PID, "note_id", args.NoteID)
		return errorResponse(id, ErrCodeInvalidParams, "Source note not found")
	}
	canReadSource, err := canReadMCPNote(ctx, env, sourceNote)
	if err != nil {
		log.Error("source note access check failed", "error", err, "path", sourceNote.Path)
		return errorResponse(id, ErrCodeInternal, "Similar search failed: "+err.Error())
	}
	if !canReadSource {
		log.Warn("source note access denied", "path", args.Path, "href", args.Href, "pid", args.PID, "note_id", args.NoteID)
		return errorResponse(id, ErrCodeInvalidParams, "Source note not found")
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
		Path:  similarSourcePath(sourceNote),
		Limit: ptr.To(int32(limit)),
	}

	results, err := similarnotes.Resolve(ctx, env, input)
	if err != nil {
		log.Error("similar search failed", "error", err, "path", input.Path)
		return errorResponse(id, ErrCodeInternal, "Similar search failed: "+err.Error())
	}

	// Format response
	var sb strings.Builder
	if len(results) == 0 {
		sb.WriteString("No similar notes found")
	} else {
		sb.WriteString(fmt.Sprintf("Found %d similar notes:\n\n", len(results)))
		for i, r := range results {
			note := r.Note.NoteView
			sb.WriteString(fmt.Sprintf("%d. %s (%.2f)\n   %s\n   %s\n\n", i+1, note.Title, r.Score, note.Path, env.NoteURL(note)))
		}
	}

	log.Debug("similar search completed", "path", input.Path, "results", len(results))

	return successResponse(id, structuredToolResult(sb.String(), buildSimilarPayload(sourceNote, results, env.NoteURL)))
}

func resolveSimilarReference(noteViews *model.NoteViews, args SimilarArguments) *model.NoteView {
	id := args.PID
	if id == 0 {
		id = args.NoteID
	}
	if id != 0 {
		return noteViews.GetByPathID(id)
	}
	if args.Path != "" {
		if note := noteViews.PathMap[args.Path]; note != nil {
			return note
		}
		return noteViews.GetByPath(args.Path)
	}
	if args.Href != "" {
		return noteViews.GetByPath(args.Href)
	}
	return nil
}

func similarSourcePath(note *model.NoteView) string {
	if note.Path != "" {
		return note.Path
	}
	return note.Permalink
}

func buildSimilarPayload(sourceNote *model.NoteView, results []graphmodel.SimilarNote, noteURL func(*model.NoteView) string) SimilarResultPayload {
	payload := SimilarResultPayload{
		Source: searchResultItemFromNote(sourceNote, 1, noteURL),
	}
	for _, r := range results {
		if r.Note == nil || r.Note.NoteView == nil {
			continue
		}
		payload.Results = append(payload.Results, searchResultItemFromNote(r.Note.NoteView, r.Score, noteURL))
	}
	return payload
}

func handleNoteHTML(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleNoteHTML")

	args, errResp := unmarshalArgs[NoteHTMLArguments](argsRaw, id, "note_html")
	if errResp != nil {
		return *errResp
	}

	if args.Path == "" && args.Href == "" && args.PID == 0 && args.NoteID == 0 {
		return errorResponse(id, ErrCodeInvalidParams, "one of pid, note_id, path, or href is required")
	}

	noteViews := env.LatestNoteViews()
	note := resolveNoteReference(noteViews, *args)
	if note == nil {
		log.Warn("note not found", "path", args.Path, "href", args.Href, "pid", args.PID, "note_id", args.NoteID)
		return errorResponse(id, ErrCodeInvalidParams, "Note not found")
	}
	canRead, err := canReadMCPNote(ctx, env, note)
	if err != nil {
		log.Error("note access check failed", "error", err, "path", note.Path)
		return errorResponse(id, ErrCodeInternal, "Note HTML failed: "+err.Error())
	}
	if !canRead {
		log.Warn("note access denied", "path", args.Path, "href", args.Href, "pid", args.PID, "note_id", args.NoteID)
		return errorResponse(id, ErrCodeInvalidParams, "Note not found")
	}

	log.Debug("note html retrieved", "path", note.Path, "pid", note.PathID)

	if args.MatchID != "" {
		if focused, ok := focusedChunkWindow(note, args.MatchID, env.LatestNoteChunks()); ok {
			return successResponse(id, textToolResult(focused))
		}
	}

	if len(args.TocPath) > 0 {
		sectionHTML := sectionHTMLByTocPath(string(note.HTML), args.TocPath)
		if sectionHTML != "" {
			return successResponse(id, textToolResult(sectionHTML))
		}
	}

	return successResponse(id, textToolResult(string(note.HTML)))
}

func focusedChunkWindow(note *model.NoteView, matchID string, chunks []model.NoteChunk) (string, bool) {
	pathID, chunkIndex, ok := parseChunkMatchID(matchID)
	if !ok || pathID != note.PathID {
		return "", false
	}

	relevant := make([]string, 0, 3)
	for _, chunk := range chunks {
		if chunk.NotePath != note.Path {
			continue
		}
		if chunk.ChunkIndex < chunkIndex-1 || chunk.ChunkIndex > chunkIndex+1 {
			continue
		}
		relevant = append(relevant, snippetFromChunk(chunk.Content, 400))
	}
	if len(relevant) == 0 {
		return "", false
	}
	return strings.Join(relevant, "\n\n"), true
}

func parseChunkMatchID(matchID string) (int64, int, bool) {
	if !strings.HasPrefix(matchID, "p") {
		return 0, 0, false
	}
	parts := strings.Split(matchID, ":c")
	if len(parts) != 2 {
		return 0, 0, false
	}
	pathID, err := strconv.ParseInt(strings.TrimPrefix(parts[0], "p"), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	chunkIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return pathID, chunkIndex, true
}

func resolveNoteReference(noteViews *model.NoteViews, args NoteHTMLArguments) *model.NoteView {
	id := args.PID
	if id == 0 {
		id = args.NoteID
	}
	if id != 0 {
		return noteViews.GetByPathID(id)
	}
	if args.Path != "" {
		if note := noteViews.PathMap[args.Path]; note != nil {
			return note
		}
	}
	if args.Href != "" {
		return noteViews.GetByPath(args.Href)
	}
	return nil
}

func handleDynamicMethod(ctx context.Context, env Env, id any, methodName string) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleDynamicMethod")

	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod != methodName {
			continue
		}
		ok, err := canReadMCPNote(ctx, env, note)
		if err != nil {
			log.Error("dynamic method access check failed", "error", err, "method", methodName)
			return errorResponse(id, ErrCodeInternal, "Access check failed: "+err.Error())
		}
		if !ok {
			log.Warn("dynamic method access denied", "method", methodName, "note_path", note.Path)
			return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+methodName)
		}
		content := string(note.Content)
		content = stripFrontmatter(content)
		log.Debug("dynamic method executed", "method", methodName, "note_path", note.Path)
		return successResponse(id, textToolResult(content))
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
	queryPrefix := env.Features().VectorSearch.Model.QueryPrefix()
	embedding, err := env.OpenAI().CreateEmbedding(ctx, queryPrefix+query)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	return vectorResultsFromChunks(
		embedding.Vector,
		env.LatestNoteChunks(),
		env.LatestNoteViews(),
		limit,
	), nil
}

type scoredChunk struct {
	chunk model.NoteChunk
	score float64
}

func vectorResultsFromChunks(
	queryEmbedding []float32,
	chunks []model.NoteChunk,
	noteViews *model.NoteViews,
	limit int,
) []model.SearchResult {
	var scores []scoredChunk
	for _, chunk := range chunks {
		if len(chunk.Embedding) == 0 {
			continue
		}
		scores = append(scores, scoredChunk{
			chunk: chunk,
			score: cosineSimilarity(queryEmbedding, chunk.Embedding),
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	results := make([]model.SearchResult, 0, len(scores))
	seen := map[string]bool{}
	for _, s := range scores {
		if seen[s.chunk.NotePath] {
			continue
		}
		seen[s.chunk.NotePath] = true

		note := noteViews.PathMap[s.chunk.NotePath]
		if note == nil || note.IsSystem() || note.ExcludeSearch {
			continue
		}
		title := note.Title
		chunkIndex := s.chunk.ChunkIndex
		results = append(results, model.SearchResult{
			NoteView:           note,
			URL:                note.Permalink,
			Score:              s.score,
			HighlightedTitle:   &title,
			HighlightedContent: []string{snippetFromChunk(s.chunk.Content, 200)},
			ChunkIndex:         &chunkIndex,
		})
		if len(results) >= limit {
			break
		}
	}

	return results
}

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

func mergeResults(textResults, vectorResults []model.SearchResult) []model.SearchResult {
	if len(vectorResults) == 0 {
		return textResults
	}

	type merged struct {
		result   model.SearchResult
		rrfScore float64
	}

	resultMap := make(map[string]*merged)

	for rank, r := range textResults {
		score := 1.0 / float64(rrfK+rank+1)
		if existing, ok := resultMap[r.URL]; ok {
			existing.rrfScore += score
		} else {
			resultMap[r.URL] = &merged{result: r, rrfScore: score}
		}
	}

	for rank, r := range vectorResults {
		score := 1.0 / float64(rrfK+rank+1)
		if existing, ok := resultMap[r.URL]; ok {
			existing.rrfScore += score
			if existing.result.ChunkIndex == nil && r.ChunkIndex != nil {
				existing.result.ChunkIndex = r.ChunkIndex
			}
			if len(existing.result.HighlightedContent) == 0 && len(r.HighlightedContent) > 0 {
				existing.result.HighlightedContent = r.HighlightedContent
			}
		} else {
			resultMap[r.URL] = &merged{result: r, rrfScore: score}
		}
	}

	var finalResults []model.SearchResult
	for _, m := range resultMap {
		m.result.Score = m.rrfScore
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

func trimWhitespace(s string) string {
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

// cosineSimilarity calculates the cosine similarity between two vectors.
// TODO: Consider replacing with Bleve's FAISS-based vector search when CGO is acceptable.
// See: https://github.com/blevesearch/bleve/blob/master/docs/vectors.md
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
