package mcp

import (
	"context"
	"fmt"

	"trip2g/internal/features"
)

// This file holds the MCP tool catalog: what tools exist, how they are
// described to agents, and which of them a given caller may see. The
// descriptions and schemas here are the contract clients consume, so they are
// kept verbatim rather than generated.

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
			Description: "Search notes by query. Returns snippets with a heading breadcrumb (title > section > subsection) that locates the approximate section, plus a precise toc_path per match. Each result carries note_path (string) and note_id (integer); each match carries match_id (string, form \"p<pid>:c<chunk>\"). Drill-down workflow: 1) search to find the approximate section via the breadcrumb; 2) call note_html(path=<result.note_path>, toc_path=[...]) to read the matched section, or expand(path=<result.note_path>, toc_path=[...]) to navigate the note's structure level by level; 3) note_html(path=<result.note_path>, match_id=<match.match_id>) for a focused chunk window. Each match also carries section_url — a link straight to that heading, for citing the section rather than the whole note.",
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
						Description: "Breadcrumb path to a specific section, e.g. [\"Chapter 1\", \"Introduction\"]. Use toc_path from a search match, or a child path from expand. Wins over match_id when both are given: match_id is only used when toc_path is absent.",
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
					"toc_path": {
						Type:        "array",
						Description: "Breadcrumb path to a specific section, e.g. [\"Chapter 1\", \"Introduction\"]. Use toc_path from a federated_search match, or a child path from federated_expand. Wins over match_id when both are given. Without either the whole note comes back.",
						Items:       &Property{Type: "string"},
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

	// The rerank argument is advertised ONLY where a cross-encoder sidecar is
	// actually configured. An argument the instance cannot honour is worse than
	// no argument: the agent burns a turn discovering that it did nothing.
	if rr := env.Features().VectorSearch.Reranker; rr.Enabled {
		rerankProp := Property{Type: "boolean", Description: rerankArgDescription(rr)}
		for i := range tools {
			if tools[i].Name == "search" || tools[i].Name == "federated_search" {
				tools[i].InputSchema.Properties["rerank"] = rerankProp
			}
		}
	}

	if env.FederatedGraphQLEnabled() {
		tools = append(tools, federatedGraphQLTool(nestedKBIDNote))
	}

	if mcpAdminToolsEnabled(ctx) {
		tools = append(tools, adminGraphQLTools()...)
	}

	return tools
}

// rerankArgDescription spells out both the effective default and the price,
// because neither reaches the model on its own: this schema has no JSON Schema
// `default` keyword, and the cost lives in the deployment, not in the protocol.
// An agent that cannot see the cost switches reranking on by reflex, and every
// search silently becomes many seconds slower — so the number is stated in the
// units the agent controls, candidates.
func rerankArgDescription(rr features.RerankerConfig) string {
	const shared = "Second-stage cross-encoder rerank: re-scores the top candidates against the query as pairs, " +
		"which orders results more accurately than the vector stage alone. " +
		"Cost is linear in candidates — this instance reranks up to %d, " +
		"which on a CPU sidecar is roughly a second each (~%ds) and on a GPU sidecar far less. "

	if rr.Default {
		return fmt.Sprintf(shared+
			"DEFAULT ON here: omit the argument to rerank, pass false to skip it when a fast answer "+
			"matters more than the ordering.", rr.TopN, rr.TopN)
	}
	return fmt.Sprintf(shared+
		"DEFAULT OFF here: pass true only when the ordering you got back looks wrong and a better one "+
		"is worth the wait. Do not set it on every call.", rr.TopN, rr.TopN)
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
