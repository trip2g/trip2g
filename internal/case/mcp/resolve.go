package mcp

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg mcp_test . Env

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/similarnotes"
	"trip2g/internal/case/sitesearch"
	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	"trip2g/internal/model"
	"trip2g/internal/openai"
	"trip2g/internal/ptr"
)

const (
	// Search and display limits.
	DefaultDisplayLimit      = 10
	DefaultSimilarLimit      = 10
	MaxSimilarLimit          = 100
	DefaultSearchLimit       = 6
	DefaultSearchDetailLimit = 3
	MaxSearchLimit           = 20

	// MCP method names.
	MCPMethodInitialize = "initialize"
	mcpMethodToolsList  = "tools/list"
	mcpMethodToolsCall  = "tools/call"
)

type Env interface {
	similarnotes.Env
	model.FederationClientFactory
	SearchLatestNotes(query string) ([]model.SearchResult, error)
	SearchLiveNotes(query string) ([]model.SearchResult, error)
	LatestNoteChunks() []model.NoteChunk
	LiveNoteChunks() []model.NoteChunk
	LiveNoteViews() *model.NoteViews
	OpenAI() *openai.Client
	SiteConfig(ctx context.Context) model.SiteConfig
	PublicURL() string
	NoteURL(note *model.NoteView) string
	Logger() logger.Logger
	FederationSecretByKBURL(ctx context.Context, kbURL string) (db.FederationSecret, bool, error)
	FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error)
	ListFederationSecretSubgraphsByKID(ctx context.Context, kid string) ([]string, error)
	DecryptData([]byte) ([]byte, error)
	FederationMaxDepth() int
	FederatedFanoutConcurrency() int
	FederatedFanoutLimit() int
	FederatedFanoutTimeout() time.Duration
	// Federated instructions cache (per kb_id path)
	CachedFederatedInstructions(kbID string) (model.FederationResult, bool)
	StoreFederatedInstructions(kbID string, result model.FederationResult)
	// API key auth
	ResolveAPIKey(ctx context.Context, value, action string) (*db.ApiKey, error)
	// Admin GraphQL tools
	GraphQLRequest(ctx context.Context, query string, variables map[string]any) ([]byte, error)
	// Federated GraphQL tools
	GraphQLRequestScoped(ctx context.Context, query string, variables map[string]any, allowedSubgraphs []string) ([]byte, error)
	FederatedGraphQLEnabled() bool
	// Prometheus metrics for the MCP endpoint; may be nil (record methods are nil-safe).
	MCPMetrics() *metrics.MCPMetrics
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

// initializeInstructions resolves the server instructions an initialize result
// carries: the content of the note whose mcp_method matches the target, with
// frontmatter stripped. methodOverride comes from ?method= and selects a note
// other than the default `initialize` one.
//
// A non-nil *Error means the request must fail outright rather than be served
// without instructions — that only happens for an explicit override naming a
// note that is missing or unreadable. A missing default initialize note is not
// an error, it just yields "".
func initializeInstructions(ctx context.Context, env Env, methodOverride string) (string, *Error) {
	target := methodOverride
	if target == "" {
		target = MCPMethodInitialize
	}

	instructions := ""
	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod != target {
			continue
		}
		ok, err := canReadMCPNote(ctx, env, note)
		if err != nil {
			return "", &Error{Code: ErrCodeInternal, Message: "Instructions access check failed: " + err.Error()}
		}
		if !ok {
			// The note exists but the caller cannot read it. For an explicit
			// ?method= override that is an error; for default initialize it is
			// a silent skip.
			break
		}
		instructions = stripFrontmatter(string(note.Content))
		break
	}

	if methodOverride != "" && instructions == "" {
		return "", &Error{Code: ErrCodeMethodNotFound, Message: "Method not found: " + methodOverride}
	}
	return instructions, nil
}

var reservedMCPTools = map[string]bool{ //nolint:gochecknoglobals // immutable set of built-in tool names
	"search":                    true,
	"similar":                   true,
	"note_html":                 true,
	"expand":                    true,
	"federated_search":          true,
	"federated_similar":         true,
	"federated_note_html":       true,
	"federated_expand":          true,
	"federated_instructions":    true,
	"graphql_introspection":     true,
	"graphql_request":           true,
	"federated_graphql_request": true,
	MCPMethodInitialize:         true,
}

// defaultKBIDNote describes kb_id for tool metadata built outside a listing,
// where the federation depth is not part of the message.
const defaultKBIDNote = "Target knowledge base id"

// builtinTools returns the tools advertised by tools/list for this request.
// Registration (registerTools) is deliberately wider: a few tools are callable
// but unlisted, so the two sets are computed separately.
func builtinTools(ctx context.Context, env Env) []Tool {
	return append(staticTools(ctx, env), dynamicTools(ctx, env)...)
}

// staticTools returns the compiled-in tools, without walking the note corpus.
func staticTools(ctx context.Context, env Env) []Tool { //nolint:funlen // flat declarative list of built-in tool schemas
	maxDepth := env.FederationMaxDepth()
	nestedKBIDNote := fmt.Sprintf(
		"Target knowledge base id; nested bases use '/' "+
			`(e.g. "philosophers/nietzsche" routes through the 'philosophers' peer, recursively). `+
			"Federation nests up to %d levels deep (kb_id path segments); a deeper path is rejected.",
		maxDepth,
	)
	tools := []Tool{
		{
			Name:        "search",
			Description: "Search notes by query. Returns snippets with a heading breadcrumb (title > section > subsection) that locates the approximate section, plus a precise toc_path per match. Each result carries note_path (string) and note_id (integer); each match carries match_id (string, form \"p<pid>:c<chunk>\"). Drill-down workflow: 1) search to find the approximate section via the breadcrumb; 2) call note_html(path=<result.note_path>, toc_path=[...]) to read the matched section, or expand(path=<result.note_path>, toc_path=[...]) to navigate the note's structure level by level; 3) note_html(path=<result.note_path>, match_id=<match.match_id>) for a focused chunk window.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query"},
					"limit": {Type: "number", Description: "Max number of results to return (default 6)"},
					"detail_limit": {
						Type:        "number",
						Description: "How many results include full snippet matches; results beyond this are returned as lightweight previews (title, path, score) to save context (default 3)",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "similar",
			Description: "Find related notes from a known note reference. Preferred: path (a search result's note_path field). Use this after opening a promising note when you need nearby context.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "String note path, e.g. \"concepts/maska-i-glubina.md\" — copy verbatim from a search result's note_path field. The default, preferred way to reference a note",
					},
					"href": {Type: "string", Description: "String note href, copied verbatim from a search result's href field"},
					"pid": {
						Type:        "number",
						Description: "Non-negative integer (uint64) note id, copied verbatim from a search result's note_id field. Not a path, slug, or match_id — a value like \":\" or \"/hub/goethe.md\" is a path, not a note id. Prefer path",
					},
					"note_id": {
						Type:        "number",
						Description: "Same as pid: non-negative integer (uint64) note id, copied verbatim from a search result's note_id field. Not a path, slug, or match_id. Prefer path",
					},
					"limit": {Type: "number", Description: "Max number of results (default 10)"},
				},
			},
		},
		{
			Name: "note_html",
			Description: "Read a note. Canonical calls, copying fields verbatim from a search result: " +
				"search(query) -> note_html(path=<result.note_path>) reads the whole note; " +
				"search(query) -> note_html(match_id=<match.match_id>) reads just the focused chunk around a hit (cheaper, targeted); " +
				"expand(path=<result.note_path>, toc_path=[...]) -> note_html(path=<result.note_path>, toc_path=[...]) reads one exact section. " +
				"Only pass pid/note_id if you already copied that exact integer from a result's note_id field — never invent one. " +
				`path is a string like "concepts/x.md"; match_id is "p<pid>:c<chunk>"; a value like ":" or "/hub/goethe.md" is a PATH, not a note_id.`,
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "String note path, e.g. \"concepts/maska-i-glubina.md\" — copy verbatim from a search result's note_path field. The default, preferred way to open a note",
					},
					"href": {Type: "string", Description: "String note href or absolute URL, copied verbatim from a search result's href field"},
					"pid": {
						Type:        "number",
						Description: "Non-negative integer (uint64) note id, copied verbatim from a search result's note_id field. Not a path, slug, or match_id — a value like \":\" or \"/hub/goethe.md\" is a path, not a note id. Prefer path or match_id",
					},
					"note_id": {
						Type:        "number",
						Description: "Same as pid: non-negative integer (uint64) note id, copied verbatim from a search result's note_id field. Not a path, slug, or match_id. Prefer path or match_id",
					},
					"match_id": {
						Type:        "string",
						Description: "String chunk id of the form \"p<pid>:c<chunk>\" (e.g. \"p32:c4\"), copied verbatim from a search match's match_id field. Alone it is enough to resolve the note and reads a focused window around that hit",
					},
					"context_words": {Type: "number", Description: "Optional future hint for expanding focused reads"},
					"toc_path": {
						Type:        "array",
						Description: "Breadcrumb path to a specific section, e.g. [\"Chapter 1\", \"Introduction\"]. Use toc_path from a search match, or a child path from expand.",
						Items:       &Property{Type: "string"},
					},
				},
			},
		},
		{
			Name:        "expand",
			Description: "Walk a note's table of contents level by level (progressive disclosure). Canonical call: expand(path=<result.note_path>, toc_path=[...]) — copy path verbatim from a search result's note_path field. Returns the direct children of a TOC node: omit toc_path (or pass []) for the top-level sections, or pass a toc_path to list that section's subsections. Each child has title, level, path, and has_children. Drill down with expand, then read a leaf with note_html(path=..., toc_path=[...]) — no need to load the whole note or its full flat TOC.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "String note path, e.g. \"concepts/maska-i-glubina.md\" — copy verbatim from a search result's note_path field. The default, preferred way to reference a note",
					},
					"href": {Type: "string", Description: "String note href, copied verbatim from a search result's href field"},
					"pid": {
						Type:        "number",
						Description: "Non-negative integer (uint64) note id, copied verbatim from a search result's note_id field. Not a path, slug, or match_id. Prefer path",
					},
					"note_id": {
						Type:        "number",
						Description: "Same as pid: non-negative integer (uint64) note id, copied verbatim from a search result's note_id field. Not a path, slug, or match_id. Prefer path",
					},
					"toc_path": {
						Type:        "array",
						Description: "Breadcrumb path to the node to expand, e.g. [\"Chapter 1\"]. Omit or [] for the top level.",
						Items:       &Property{Type: "string"},
					},
				},
			},
		},
		{
			Name: "federated_search",
			Description: fmt.Sprintf(
				"Search connected knowledge bases. Returns snippets with heading breadcrumbs (title > section > subsection) and a precise toc_path per match, same as search; results also carry an absolute kb_id (string) to use verbatim on follow-up calls. Pass kb_id for one base, kb_ids for selected bases, or omit both to fan out. Nested bases are addressed with '/': kb_id \"philosophers/nietzsche\" routes through the 'philosophers' peer to the base it federates (recursive), up to %d levels deep. Canonical call: federated_search(kb_id=\"philosophers/<author>\", query) -> federated_note_html(kb_id=\"philosophers/<author>\", path=<result.note_path>) — the standard way to descend into a leaf corpus and read real content, not hub cards.",
				maxDepth,
			),
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query"},
					"kb_id": {
						Type:        "string",
						Description: nestedKBIDNote,
					},
					"kb_ids": {
						Type: "array",
						Description: "Target knowledge base ids; each accepts the same " +
							"nested 'peer/base' form as kb_id",
						Items: &Property{Type: "string"},
					},
					"limit": {Type: "number", Description: "Max number of results to return (default 6)"},
					"detail_limit": {
						Type:        "number",
						Description: "How many results include full snippet matches; results beyond this are returned as lightweight previews (title, path, score) to save context (default 3)",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "federated_similar",
			Description: "Find remote notes similar to a known note reference inside a connected knowledge base. Preferred: path (a federated_search result's note_path field).",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"kb_id": {
						Type:        "string",
						Description: nestedKBIDNote,
					},
					"path": {
						Type:        "string",
						Description: "String remote note path, copied verbatim from a federated_search result's note_path field",
					},
					"href": {Type: "string", Description: "String remote note href, copied verbatim from a federated_search result's href field"},
					"pid": {
						Type:        "number",
						Description: "Non-negative integer (uint64) remote note id, copied verbatim from a federated_search result's note_id field. Prefer path",
					},
					"note_id": {
						Type:        "string",
						Description: "Same remote note id as pid, but as a STRING (uint64) — stringify the federated_search result's note_id field. Prefer path",
					},
					"limit": {Type: "number", Description: "Max number of results"},
				},
				Required: []string{"kb_id"},
			},
		},
		{
			Name: "federated_note_html",
			Description: "Read a remote note inside a connected knowledge base. Canonical call, copying fields verbatim from a federated_search result: " +
				"federated_search(kb_id=\"philosophers/<author>\", query) -> federated_note_html(kb_id=\"philosophers/<author>\", path=<result.note_path>) — " +
				"the standard way to descend into a leaf corpus and read real content there, not hub cards. " +
				"federated_note_html(kb_id=..., match_id=<match.match_id>) reads just the focused chunk around a hit. " +
				"Only pass pid/note_id if you already copied that exact id from a result. " +
				`path is a string like "concepts/x.md"; match_id is "p<pid>:c<chunk>"; a value like ":" or "/hub/goethe.md" is a PATH, not a note_id.`,
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"kb_id": {
						Type:        "string",
						Description: nestedKBIDNote,
					},
					"path": {
						Type:        "string",
						Description: "String remote note path, copied verbatim from a federated_search result's note_path field",
					},
					"href": {
						Type:        "string",
						Description: "String remote note href or absolute URL, copied verbatim from a federated_search result's href field",
					},
					"pid": {
						Type:        "number",
						Description: "Non-negative integer (uint64) remote note id, copied verbatim from a federated_search result's note_id field. Not a path, slug, or match_id. Prefer path or match_id",
					},
					"note_id": {
						Type:        "string",
						Description: "Same remote note id as pid, but as a STRING (uint64) — stringify the federated_search result's note_id field. Not a path, slug, or match_id. Prefer path or match_id",
					},
					"match_id": {
						Type:        "string",
						Description: "String chunk id of the form \"p<pid>:c<chunk>\", copied verbatim from a remote search match's match_id field; alone it is enough to resolve the note",
					},
				},
				Required: []string{"kb_id"},
			},
		},
		{
			Name:        "federated_expand",
			Description: "Walk a remote note's table of contents level by level inside a connected knowledge base (progressive disclosure), same as expand. Canonical call: federated_expand(kb_id=..., path=<result.note_path>, toc_path=[...]). Omit toc_path for the top level, or pass a toc_path to list that node's subsections.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"kb_id": {
						Type:        "string",
						Description: nestedKBIDNote,
					},
					"path": {
						Type:        "string",
						Description: "String remote note path, copied verbatim from a federated_search result's note_path field",
					},
					"href": {Type: "string", Description: "String remote note href, copied verbatim from a federated_search result's href field"},
					"pid": {
						Type:        "number",
						Description: "Non-negative integer (uint64) remote note id, copied verbatim from a federated_search result's note_id field. Prefer path",
					},
					"note_id": {
						Type:        "string",
						Description: "Same remote note id as pid, but as a STRING (uint64) — stringify the federated_search result's note_id field. Prefer path",
					},
					"toc_path": {
						Type:        "array",
						Description: "Breadcrumb path to the node to expand. Omit or [] for the top level.",
						Items:       &Property{Type: "string"},
					},
				},
				Required: []string{"kb_id"},
			},
		},
		{
			Name: "federated_instructions",
			Description: "Fetch the instructions/guidance for a federated knowledge base by kb_id " +
				`(e.g. "philosophers/nietzsche") — read a base's own conventions before searching it. ` +
				"Nested bases are addressed with '/' and the call routes through each peer recursively.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"kb_id": {
						Type: "string",
						Description: "Target knowledge base id; nested bases use '/' " +
							`(e.g. "philosophers/nietzsche" routes through the 'philosophers' peer, recursively)`,
					},
				},
				Required: []string{"kb_id"},
			},
		},
	}

	if env.FederatedGraphQLEnabled() {
		tools = append(tools, federatedGraphQLTool(nestedKBIDNote))
	}

	if mcpAdminToolsEnabled(ctx) {
		tools = append(tools, adminGraphQLTools()...)
	}

	return tools
}

// dynamicTools lists the tools notes register through mcp_method frontmatter.
//
// Several notes can share an mcp_method (e.g. localized en/ru instruction
// notes) — handleDynamicMethod resolves to the first note in path-sorted order,
// so each method is listed once, keeping that same first note. Otherwise
// tools/list would return duplicate tool names.
func dynamicTools(ctx context.Context, env Env) []Tool {
	var tools []Tool
	seenMCPMethods := make(map[string]bool)
	for _, note := range env.LatestNoteViews().List {
		if note.MCPMethod == "" || reservedMCPTools[note.MCPMethod] {
			continue
		}
		if seenMCPMethods[note.MCPMethod] {
			continue
		}
		// Claim the method for this note before the read check so the first note
		// in path-sorted order always owns it — matching handleDynamicMethod,
		// which stops at that same first note whether or not it is readable.
		seenMCPMethods[note.MCPMethod] = true
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
	return tools
}

// toolHandler is the shape shared by every built-in tool implementation.
type toolHandler func(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response

// builtinToolHandlers maps a tool name to its implementation. The set is wider
// than what tools/list advertises: authorization is enforced per tool (either
// here or inside the handler), while listing is filtered separately.
func builtinToolHandlers() map[string]toolHandler { //nolint:gocognit // flat dispatch table
	return map[string]toolHandler{
		"search":                    handleSearch,
		"similar":                   handleSimilar,
		"note_html":                 handleNoteHTML,
		"expand":                    handleExpand,
		"federated_search":          handleFederatedSearch,
		"federated_similar":         handleFederatedSimilar,
		"federated_note_html":       handleFederatedNoteHTML,
		"federated_expand":          handleFederatedExpand,
		"federated_instructions":    handleFederatedInstructions,
		"federated_graphql_request": handleFederatedGraphQLRequest,
		"graphql_introspection": func(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
			if !mcpAdminToolsEnabled(ctx) {
				return errorResponse(id, ErrCodeMethodNotFound, "Method not found: graphql_introspection")
			}
			return handleGraphQLIntrospection(ctx, env, id, argsRaw)
		},
		// graphql_request serves two callers: a federation peer gets a
		// subgraph-scoped executor, an admin API key gets the full one.
		"graphql_request": func(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
			if fedAuth, ok := federationAuthFromContext(ctx); ok {
				if !env.FederatedGraphQLEnabled() {
					return errorResponse(id, ErrCodeMethodNotFound, "Method not found: graphql_request")
				}
				return handleGraphQLRequestScoped(ctx, env, id, argsRaw, fedAuth.AllowedSubgraphs)
			}
			if !mcpAdminToolsEnabled(ctx) {
				return errorResponse(id, ErrCodeMethodNotFound, "Method not found: graphql_request")
			}
			return handleGraphQLRequest(ctx, env, id, argsRaw)
		},
	}
}

func federatedGraphQLTool(nestedKBIDNote string) Tool {
	return Tool{
		Name:        "federated_graphql_request",
		Description: "Forwards a read-only GraphQL query to a federation peer KB. Scoped to the caller's allowed subgraphs.",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"kb_id":     {Type: "string", Description: nestedKBIDNote},
				"query":     {Type: "string", Description: "Read-only GraphQL query string"},
				"variables": {Type: "object", Description: "Optional variables map"},
			},
			Required: []string{"kb_id", "query"},
		},
	}
}

func adminGraphQLTools() []Tool {
	return []Tool{
		{
			Name:        "graphql_introspection",
			Description: "Inspect the GraphQL schema. Returns types and operations matching the pattern (regexp), plus all types they reference. Use this to discover available mutations and queries before calling graphql_request.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern": {Type: "string", Description: "Regexp or substring to filter type and operation names"},
				},
				Required: []string{"pattern"},
			},
		},
		{
			Name:        "graphql_request",
			Description: "Execute a GraphQL query or mutation as admin. Use graphql_introspection first to find the right operation.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":     {Type: "string", Description: "GraphQL query or mutation string"},
					"variables": {Type: "object", Description: "Optional variables map"},
				},
				Required: []string{"query"},
			},
		},
	}
}

// resolveSearchLimits normalises and clamps limit and detailLimit following the
// same pattern as handleSimilar's limit handling.
func resolveSearchLimits(log logger.Logger, limit, detailLimit int) (int, int) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	} else if limit > MaxSearchLimit {
		log.Warn("search limit exceeds maximum, capping", "requested", limit, "max", MaxSearchLimit)
		limit = MaxSearchLimit
	}
	if detailLimit <= 0 {
		detailLimit = DefaultSearchDetailLimit
	}
	if detailLimit > limit {
		detailLimit = limit
	}
	return limit, detailLimit
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

	// Shared retrieval core (text + vector + RRF + rerank), same as the site
	// search, and the same corpus rule: instances that show draft versions
	// search latest for everyone, API-key clients always get latest, everyone
	// else gets the live corpus, like anonymous site visitors.
	siteConfig := env.SiteConfig(ctx)
	useLatest := siteConfig.ShowDraftVersions || mcpAPIKeyAuthed(ctx)
	results, _, err := sitesearch.Retrieve(ctx, env, args.Query, useLatest)
	if err != nil {
		log.Error("search failed", "error", err, "query", args.Query)
		return errorResponse(id, ErrCodeInternal, "Search failed: "+err.Error())
	}

	results, err = filterSearchResults(ctx, env, results)
	if err != nil {
		log.Error("search access check failed", "error", err, "query", args.Query)
		return errorResponse(id, ErrCodeInternal, "Search failed: "+err.Error())
	}

	chunks := env.LiveNoteChunks()
	if useLatest {
		chunks = env.LatestNoteChunks()
	}

	limit, detailLimit := resolveSearchLimits(log, args.Limit, args.DetailLimit)
	payload := buildSearchPayload(args.Query, results, env.NoteURL, chunks, limit, detailLimit)
	metricsFromContext(ctx).ObserveSearchResults("search", len(payload.Results))

	// Format response
	var sb strings.Builder
	if len(payload.Results) == 0 {
		sb.WriteString("No results found for: " + args.Query)
	} else {
		sb.WriteString(fmt.Sprintf("Found %d notes:\n\n", len(payload.Results)))
		for i, r := range payload.Results {
			sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n", i+1, r.Title, r.NotePath, r.URL))
			if len(r.Matches) > 0 {
				sb.WriteString(fmt.Sprintf("   %s\n", r.Matches[0].Snippet))
				if r.Matches[0].MatchID != "" {
					sb.WriteString(fmt.Sprintf("   match_id: %s\n", r.Matches[0].MatchID))
				}
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

func buildSearchPayload(
	query string,
	results []model.SearchResult,
	noteURL func(*model.NoteView) string,
	chunks []model.NoteChunk,
	limit, detailLimit int,
) SearchResultPayload {
	payload := SearchResultPayload{Query: query}
	count := 0
	for _, r := range results {
		if r.NoteView == nil {
			continue
		}
		if count >= limit {
			break
		}

		item := searchResultItemFromNote(r.NoteView, r.Score, noteURL)
		if count < detailLimit {
			// Full detail: include snippet Matches.
			for _, snippet := range r.HighlightedContent {
				chunkIndex := -1 // -1 = no chunk resolved (chunk indices are 0-based)
				if r.ChunkIndex != nil {
					chunkIndex = *r.ChunkIndex
				} else if nearest, ok := nearestChunkIndexForSnippet(r.NoteView, snippet, chunks); ok {
					chunkIndex = nearest
				}
				match := SearchMatch{
					Snippet:      snippet,
					ContextWords: 10,
				}
				if chunkIndex >= 0 {
					// Only chunk-based ids: note_html can't resolve anything else,
					// so an unresolved snippet gets no match_id at all.
					match.MatchID = fmt.Sprintf("p%d:c%d", r.NoteView.PathID, chunkIndex)
					match.ChunkIndex = chunkIndex
				}
				chunkContent := chunkContentByIndex(r.NoteView, chunkIndex, chunks)
				match.TOCPath = tocPathForSnippet(string(r.NoteView.HTML), snippet, chunkContent)
				item.Matches = append(item.Matches, match)
			}
		}
		// Beyond detailLimit: item.Matches stays nil (lightweight preview).
		payload.Results = append(payload.Results, item)
		count++
	}
	return payload
}

// chunkContentByIndex returns the Content of the chunk for the given note at
// chunkIndex, or "" when chunkIndex is negative (meaning no chunk was resolved)
// or the chunk is not found. Chunk indices are 0-based.
func chunkContentByIndex(note *model.NoteView, chunkIndex int, chunks []model.NoteChunk) string {
	if note == nil || chunkIndex < 0 || len(chunks) == 0 {
		return ""
	}
	for _, chunk := range chunks {
		if chunk.NotePath == note.Path && chunk.ChunkIndex == chunkIndex {
			return chunk.Content
		}
	}
	return ""
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
		normalizedChunk := normalizeSearchSnippet(sitesearch.SnippetFromChunk(chunk.Content, 400))
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
	s = sitesearch.TrimWhitespace(strings.ToLower(s))
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
	}
	if kb := model.NewMCPFederationNote(note); kb != nil {
		item.Kind = "federation_kb"
		// A pointer note's own address, relative to this hub. Federation hops
		// prefix their local segment onto it on the way back to the caller.
		item.KBID = kb.ID
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
		`This is a knowledge base pointer. To search inside it, call federated_search with kb_id="%s". To open notes from it, call federated_note_html(path=..., kb_id="%s").`,
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
	metricsFromContext(ctx).ObserveSearchResults("similar", len(results))

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

	if args.Path == "" && args.Href == "" && args.PID.Value == 0 && args.PID.Raw == "" &&
		args.NoteID.Value == 0 && args.NoteID.Raw == "" && args.MatchID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "one of pid, note_id, path, href, or match_id is required")
	}

	noteViews := env.LatestNoteViews()
	note := resolveNoteReference(noteViews, *args)
	if note == nil {
		log.Warn("note not found", "path", args.Path, "href", args.Href, "pid", args.PID.Value, "pid_raw", args.PID.Raw, "note_id", args.NoteID.Value)
		if args.PID.Raw != "" && args.PID.Value == 0 {
			return errorResponse(id, ErrCodeInvalidParams,
				fmt.Sprintf("pid %q is not a note id; note ids are numbers from search results — chunk refs like \"p36:c2\" go in match_id", args.PID.Raw))
		}
		return errorResponse(id, ErrCodeInvalidParams, "Note not found")
	}
	canRead, err := canReadMCPNote(ctx, env, note)
	if err != nil {
		log.Error("note access check failed", "error", err, "path", note.Path)
		return errorResponse(id, ErrCodeInternal, "Note HTML failed: "+err.Error())
	}
	if !canRead {
		log.Warn("note access denied", "path", args.Path, "href", args.Href, "pid", args.PID.Value, "note_id", args.NoteID.Value)
		return errorResponse(id, ErrCodeInvalidParams, "Note not found")
	}

	log.Debug("note html retrieved", "path", note.Path, "pid", note.PathID)

	// A pointer miss must never silently dump the full note: that converts a
	// ~300-token read into the whole-note anti-pattern. Fail loud and cheap.
	if args.MatchID != "" {
		if focused, ok := focusedChunkWindow(note, args.MatchID, env.LatestNoteChunks()); ok {
			return successResponse(id, textToolResult(focused))
		}
		if len(args.TocPath) == 0 {
			log.Warn("match_id did not resolve", "path", note.Path, "match_id", args.MatchID)
			return errorResponse(id, ErrCodeInvalidParams,
				fmt.Sprintf("no focused window for match_id %q; %s", args.MatchID, topLevelSectionsNudge(note)))
		}
	}

	if len(args.TocPath) > 0 {
		sectionHTML := sectionHTMLByTocPath(string(note.HTML), args.TocPath)
		if sectionHTML != "" {
			return successResponse(id, textToolResult(sectionHTML))
		}
		log.Warn("toc_path did not resolve", "path", note.Path, "toc_path", args.TocPath)
		return errorResponse(id, ErrCodeInvalidParams,
			fmt.Sprintf("section not found for toc_path [%s]; %s", strings.Join(args.TocPath, " > "), topLevelSectionsNudge(note)))
	}

	// A resolved note with no rendered content must fail loud: silent empty
	// success poisons downstream consumers that treat text:"" as the note body.
	if len(note.HTML) == 0 {
		log.Warn("note has no rendered content", "path", note.Path, "pid", note.PathID)
		return errorResponse(id, ErrCodeInvalidParams,
			fmt.Sprintf("note %q resolved but has no rendered content", note.Path))
	}

	return successResponse(id, textToolResult(string(note.HTML)))
}

// topLevelSectionsNudge is the cheap (~30 token) expand-shaped hint returned on
// a pointer miss instead of the full note.
func topLevelSectionsNudge(note *model.NoteView) string {
	children := tocChildren(note.Headings, nil)
	if len(children) == 0 {
		return "the note has no sections; call note_html without toc_path to read the whole note"
	}
	titles := make([]string, 0, len(children))
	for _, c := range children {
		titles = append(titles, c.Title)
	}
	return "top-level sections: " + strings.Join(titles, ", ") + " — navigate with expand or note_html(toc_path=[...])"
}

func handleExpand(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	log := logger.WithPrefix(env.Logger(), "mcp:handleExpand")

	args, errResp := unmarshalArgs[ExpandArguments](argsRaw, id, "expand")
	if errResp != nil {
		return *errResp
	}

	if args.Path == "" && args.Href == "" && args.PID == 0 && args.NoteID == 0 {
		return errorResponse(id, ErrCodeInvalidParams, "one of pid, note_id, path, or href is required")
	}

	noteViews := env.LatestNoteViews()
	note := resolveNoteReference(noteViews, NoteHTMLArguments{
		Path:   args.Path,
		Href:   args.Href,
		PID:    PID{Value: args.PID},
		NoteID: PID{Value: args.NoteID},
	})
	if note == nil {
		log.Warn("note not found", "path", args.Path, "href", args.Href, "pid", args.PID, "note_id", args.NoteID)
		return errorResponse(id, ErrCodeInvalidParams, "Note not found")
	}
	canRead, err := canReadMCPNote(ctx, env, note)
	if err != nil {
		log.Error("note access check failed", "error", err, "path", note.Path)
		return errorResponse(id, ErrCodeInternal, "Expand failed: "+err.Error())
	}
	if !canRead {
		log.Warn("note access denied", "path", args.Path, "href", args.Href, "pid", args.PID, "note_id", args.NoteID)
		return errorResponse(id, ErrCodeInvalidParams, "Note not found")
	}

	children := tocChildren(note.Headings, args.TocPath)
	payload := ExpandPayload{
		NoteID:   note.PathID,
		NotePath: note.Path,
		TocPath:  args.TocPath,
		Children: children,
	}
	log.Debug("expand completed", "path", note.Path, "toc_path", args.TocPath, "children", len(children))
	return successResponse(id, structuredToolResult(expandSummary(note, args.TocPath, children), payload))
}

// expandSummary renders a short human-readable view of an expand result for the
// text content block; the structured payload carries the machine-readable tree.
func expandSummary(note *model.NoteView, parentPath []string, children []TOCNode) string {
	where := "top level"
	if len(parentPath) > 0 {
		where = strings.Join(parentPath, " > ")
	}
	var sb strings.Builder
	if len(children) == 0 {
		fmt.Fprintf(&sb, "%s — %q has no subsections (leaf). Read it with note_html(toc_path).", note.Title, where)
		return sb.String()
	}
	fmt.Fprintf(&sb, "%s — %q, %d subsection(s):\n", note.Title, where, len(children))
	for _, c := range children {
		marker := ""
		if c.HasChildren {
			marker = " (has subsections)"
		}
		fmt.Fprintf(&sb, "- %s%s\n", c.Title, marker)
	}
	return sb.String()
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
		relevant = append(relevant, sitesearch.SnippetFromChunk(chunk.Content, 400))
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
	id := args.PID.Value
	if id == 0 {
		id = args.NoteID.Value
	}
	if id != 0 {
		if note := noteViews.GetByPathID(id); note != nil {
			return note
		}
		// Models routinely replay a stale or foreign id as pid alongside a
		// valid path — a pid miss must not eclipse the other references.
	}
	if args.Path != "" {
		if note := noteViews.PathMap[args.Path]; note != nil {
			return note
		}
	}
	if args.Href != "" {
		if note := noteViews.GetByPath(normalizeHref(args.Href)); note != nil {
			return note
		}
	}
	// A match_id like "p12:c0" carries the note id — search results hand it
	// back as the primary pointer, so it must resolve on its own.
	if pathID, _, ok := parseChunkMatchID(args.MatchID); ok {
		return noteViews.GetByPathID(pathID)
	}
	return nil
}

// normalizeHref reduces an absolute URL (the url field of search results) to
// its path component so it resolves like a relative href.
func normalizeHref(href string) string {
	if !strings.Contains(href, "://") {
		return href
	}
	u, err := url.Parse(href)
	if err != nil || u.Path == "" {
		return href
	}
	return u.Path
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

func errorResponse(id any, code int, message string) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}
