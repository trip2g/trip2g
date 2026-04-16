package mcp

import "encoding/json"

// JSON-RPC 2.0 types

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id"`
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
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
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
	Query string `json:"query"`
}

type SearchResultPayload struct {
	Query   string             `json:"query"`
	Results []SearchResultItem `json:"results"`
}

type SearchResultItem struct {
	Title    string        `json:"title"`
	NoteID   int64         `json:"note_id"`
	NotePath string        `json:"note_path"`
	Href     string        `json:"href"`
	URL      string        `json:"url"`
	Kind     string        `json:"kind"`
	Score    float64       `json:"score"`
	Matches  []SearchMatch `json:"matches,omitempty"`
}

type SearchMatch struct {
	MatchID      string       `json:"match_id"`
	ChunkIndex   int          `json:"chunk_index,omitempty"`
	Snippet      string       `json:"snippet"`
	ContextWords int          `json:"context_words"`
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
	Path         string `json:"path,omitempty"`
	Href         string `json:"href,omitempty"`
	PID          int64  `json:"pid,omitempty"`
	NoteID       int64  `json:"note_id,omitempty"`
	MatchID      string `json:"match_id,omitempty"`
	ContextWords int    `json:"context_words,omitempty"`
}

// JSON-RPC error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)
