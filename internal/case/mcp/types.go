package mcp

import (
	"encoding/json"

	"trip2g/internal/model"
)

// JSON-RPC 2.0 types

type Request struct {
	JSONRPC        string          `json:"jsonrpc"`
	Method         string          `json:"method"`
	Params         json.RawMessage `json:"params,omitempty"`
	ID             any             `json:"id"`
	MethodOverride string          `json:"-"` // from ?method= query param, overrides initialize instructions source
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
	ID      any    `json:"id"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP Protocol types

type Tool struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	InputSchema *InputSchema `json:"inputSchema,omitempty"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string    `json:"type"`
	Description string    `json:"description,omitempty"`
	Items       *Property `json:"items,omitempty"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content           []Content `json:"content"`
	StructuredContent any       `json:"structuredContent,omitempty"`
	IsError           bool      `json:"isError,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Tool-specific argument types

type SearchArguments struct {
	Query       string `json:"query"`
	Limit       int    `json:"limit,omitempty"`
	DetailLimit int    `json:"detail_limit,omitempty"`
}

type FederatedSearchArguments struct {
	Query       string   `json:"query"`
	KBID        string   `json:"kb_id,omitempty"`
	KBIDs       []string `json:"kb_ids,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	DetailLimit int      `json:"detail_limit,omitempty"`
}

type FederatedSimilarArguments struct {
	KBID   string `json:"kb_id"`
	PID    int64  `json:"pid,omitempty"`
	NoteID string `json:"note_id,omitempty"`
	Path   string `json:"path,omitempty"`
	Href   string `json:"href,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type FederatedNoteHTMLArguments struct {
	KBID    string `json:"kb_id"`
	PID     int64  `json:"pid,omitempty"`
	NoteID  string `json:"note_id,omitempty"`
	Path    string `json:"path,omitempty"`
	Href    string `json:"href,omitempty"`
	MatchID string `json:"match_id,omitempty"`
}

type FederatedExpandArguments struct {
	KBID    string   `json:"kb_id"`
	PID     int64    `json:"pid,omitempty"`
	NoteID  string   `json:"note_id,omitempty"`
	Path    string   `json:"path,omitempty"`
	Href    string   `json:"href,omitempty"`
	TocPath []string `json:"toc_path,omitempty"`
}

type FederationRef struct {
	KBID             string `json:"kb_id"`
	KBURL            string `json:"kb_url"`
	AgentInstruction string `json:"agent_instruction"`
}

type PayloadContext struct {
	KBInstructions map[string]string `json:"kb_instructions,omitempty"`
}

type FederationStatusPayload struct {
	Status  string `json:"status"`
	KBID    string `json:"kb_id,omitempty"`
	Message string `json:"message,omitempty"`
}

type FederatedCallPayload struct {
	Results []FederatedCallResult `json:"results,omitempty"`
	Errors  []FederatedCallError  `json:"errors,omitempty"`
}

type FederatedCallResult struct {
	KBID    string                 `json:"kb_id"`
	Result  model.FederationResult `json:"result"`
	Latency string                 `json:"latency"`
}

type FederatedCallError struct {
	KBID  string `json:"kb_id"`
	Error string `json:"error"`
}

type SearchResultPayload struct {
	Query   string             `json:"query"`
	Results []SearchResultItem `json:"results"`
}

type SearchResultItem struct {
	Title      string         `json:"title"`
	NoteID     int64          `json:"note_id"`
	NotePath   string         `json:"note_path"`
	Href       string         `json:"href"`
	URL        string         `json:"url"`
	Kind       string         `json:"kind"`
	Score      float64        `json:"score"`
	Matches    []SearchMatch  `json:"matches,omitempty"`
	Federation *FederationRef `json:"federation,omitempty"`
}

type SearchMatch struct {
	MatchID      string       `json:"match_id"`
	ChunkIndex   int          `json:"chunk_index,omitempty"`
	Snippet      string       `json:"snippet"`
	ContextWords int          `json:"context_words"`
	TOCPath      []string     `json:"toc_path,omitempty"`
	Links        []SearchLink `json:"links,omitempty"`
}

type SearchLink struct {
	Label  string `json:"label"`
	NoteID int64  `json:"note_id"`
	Href   string `json:"href"`
}

type SimilarArguments struct {
	Path   string `json:"path,omitempty"`
	Href   string `json:"href,omitempty"`
	PID    int64  `json:"pid,omitempty"`
	NoteID int64  `json:"note_id,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type SimilarResultPayload struct {
	Source  SearchResultItem   `json:"source"`
	Results []SearchResultItem `json:"results"`
}

type NoteHTMLArguments struct {
	Path         string   `json:"path,omitempty"`
	Href         string   `json:"href,omitempty"`
	PID          int64    `json:"pid,omitempty"`
	NoteID       int64    `json:"note_id,omitempty"`
	MatchID      string   `json:"match_id,omitempty"`
	ContextWords int      `json:"context_words,omitempty"`
	TocPath      []string `json:"toc_path,omitempty"`
}

type ExpandArguments struct {
	Path    string   `json:"path,omitempty"`
	Href    string   `json:"href,omitempty"`
	PID     int64    `json:"pid,omitempty"`
	NoteID  int64    `json:"note_id,omitempty"`
	TocPath []string `json:"toc_path,omitempty"`
}

// TOCNode is one node in a progressive-disclosure TOC walk: a direct child of the
// node addressed by the request's toc_path. HasChildren tells an agent whether it
// can drill further (expand) or should read the leaf (note_html).
type TOCNode struct {
	Title       string   `json:"title"`
	Level       int      `json:"level"`
	Path        []string `json:"path"`
	HasChildren bool     `json:"has_children"`
}

type ExpandPayload struct {
	NoteID   int64     `json:"note_id"`
	NotePath string    `json:"note_path"`
	TocPath  []string  `json:"toc_path,omitempty"`
	Children []TOCNode `json:"children"`
}

type GraphQLIntrospectionArguments struct {
	Pattern string `json:"pattern"`
}

type GraphQLRequestArguments struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type FederatedGraphQLRequestArguments struct {
	KBID      string         `json:"kb_id"`
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// JSON-RPC error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)
