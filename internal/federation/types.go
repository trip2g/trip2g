package federation

//go:generate go tool github.com/mailru/easyjson/easyjson -all -no_std_marshalers ./types.go

import "encoding/json"

type rpcRequest struct {
	JSONRPC string     `json:"jsonrpc"`
	Method  string     `json:"method"`
	Params  toolParams `json:"params"`
	ID      string     `json:"id"`
}

type toolParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments,omitempty"`
}

type rpcResponse struct {
	Result rpcResult `json:"result"`
	Error  *rpcError `json:"error,omitempty"`
}

// rpcResult is the raw MCP result envelope; StructuredContent is kept as
// raw JSON to avoid a double-decode on the federation path.
type rpcResult struct {
	Content           []rpcContent    `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
	IsError           bool            `json:"isError,omitempty"`
}

type rpcContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
