package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// ResolveForTest runs a single JSON-RPC message through the MCP server exactly
// as Endpoint.Handle does once authentication has been resolved, and decodes
// the transport's answer back into a Response.
//
// It exists because the context is the unit under test in most of this
// package's cases: API-key admin rights, federation identity and hop depth all
// travel in ctx, and driving them through a real fasthttp request would say
// more about header parsing than about the behaviour being asserted.
func ResolveForTest(ctx context.Context, env Env, req Request) Response {
	instructions := ""
	if req.Method == MCPMethodInitialize {
		resolved, instrErr := initializeInstructions(ctx, env, req.MethodOverride)
		if instrErr != nil {
			return Response{JSONRPC: "2.0", ID: req.ID, Error: instrErr}
		}
		instructions = resolved
	}

	if errResp := checkToolCallParams(req); errResp != nil {
		return *errResp
	}
	if len(req.Params) == 0 && req.Method == MCPMethodInitialize {
		req.Params = json.RawMessage(`{}`)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, "Internal Error")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "/_system/mcp", bytes.NewReader(body))
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, "Internal Error")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	rec := serveMessage(ctx, env, instructions, toolScope(req), httpReq)

	if bytes.HasPrefix(rec.body, []byte("JSON RPC not handled:")) {
		return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: "+req.Method)
	}

	var resp Response
	if json.Unmarshal(rec.body, &resp) != nil {
		return Response{}
	}
	// Decoding turns every id into a float64; JSON-RPC echoes the request's id,
	// so restore the caller's own value and keep assertions on it typed.
	if resp.ID != nil {
		resp.ID = req.ID
	}
	if resp.Error != nil {
		resp.Error.Message = methodNotFoundMessage(resp.Error, req)
		return resp
	}
	resp.Result = typedResult(req.Method, rec.body)
	return resp
}

// typedResult re-decodes the result into the concrete type the dispatcher used
// to hand back, so assertions can keep type-asserting instead of digging
// through map[string]any.
func typedResult(method string, body []byte) any {
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal(body, &envelope) != nil || len(envelope.Result) == 0 {
		return nil
	}

	switch method {
	case mcpMethodToolsList:
		var result ListToolsResult
		if json.Unmarshal(envelope.Result, &result) != nil {
			return nil
		}
		return result
	case mcpMethodToolsCall:
		var result CallToolResult
		if json.Unmarshal(envelope.Result, &result) != nil {
			return nil
		}
		// StructuredContent arrives as JSON; keep it raw so callers decode it
		// into whichever payload type the tool documents.
		var raw struct {
			StructuredContent json.RawMessage `json:"structuredContent"`
		}
		if json.Unmarshal(envelope.Result, &raw) == nil && len(raw.StructuredContent) > 0 {
			result.StructuredContent = raw.StructuredContent
		}
		return result
	default:
		var result map[string]any
		if json.Unmarshal(envelope.Result, &result) != nil {
			return nil
		}
		return result
	}
}

// ContextWithAPIKeyAuthForTest marks ctx as authenticated by an API key,
// optionally carrying the admin-tools grant.
func ContextWithAPIKeyAuthForTest(ctx context.Context, adminTools bool) context.Context {
	return contextWithMCPAPIKeyAuth(ctx, adminTools)
}

// ContextWithFederationAuthForTest marks ctx as an authenticated federation
// peer scoped to the given subgraphs.
func ContextWithFederationAuthForTest(ctx context.Context, kid string, allowedSubgraphs []string) context.Context {
	return contextWithFederationAuth(ctx, kid, allowedSubgraphs)
}

// ContextWithFederationDepthForTest stamps the inbound hop depth onto ctx.
func ContextWithFederationDepthForTest(ctx context.Context, depth int) context.Context {
	return contextWithFederationDepth(ctx, depth)
}

// BuiltinToolsForTest returns the tools tools/list advertises for this caller.
func BuiltinToolsForTest(ctx context.Context, env Env) []Tool {
	return builtinTools(ctx, env)
}
