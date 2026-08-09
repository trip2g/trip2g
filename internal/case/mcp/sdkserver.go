package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "trip2g-mcp"
	serverVersion = "1.0.0"
)

// buildServer assembles the per-request MCP server. It is per-request because
// both the tool set and the instructions depend on who is calling: an admin API
// key sees the GraphQL tools, a federation peer gets a scoped graphql_request,
// and dynamic note tools are filtered by the caller's read access.
// scope names the single tool a request needs registered. A tools/call only
// dispatches one tool, so registering the whole catalog — which means walking
// every note and access-checking each mcp_method one — would put work on the
// hot path that the endpoint never used to do. An empty scope registers
// everything, which is what tools/list needs.
func buildServer(ctx context.Context, env Env, instructions, scope string) *mcpsdk.Server {
	srv := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: serverName, Version: serverVersion},
		&mcpsdk.ServerOptions{Instructions: instructions},
	)

	registerTools(ctx, srv, env, scope)
	if scope == "" {
		srv.AddReceivingMiddleware(listedToolsMiddleware(ctx, env))
	}
	srv.AddReceivingMiddleware(unknownToolMiddleware())

	return srv
}

// registerTools registers the callable tools in scope. The callable set is
// deliberately wider than what tools/list advertises — graphql_request stays
// callable by a federation peer that never sees it listed — so the listing is
// narrowed separately by listedToolsMiddleware.
func registerTools(ctx context.Context, srv *mcpsdk.Server, env Env, scope string) {
	handlers := builtinToolHandlers()

	if scope == scopeCapabilities {
		for name := range handlers {
			srv.AddTool(&mcpsdk.Tool{Name: name, InputSchema: &jsonschema.Schema{Type: "object"}}, noopToolHandler)
		}
		return
	}

	if scope != "" {
		// Only the named tool can be dispatched, and a call never echoes tool
		// metadata, so the descriptive catalog is not built here. Arguments are
		// validated by the handlers themselves (AddTool leaves that to the
		// caller), and an unknown name falls through to handleDynamicMethod,
		// which resolves the backing note and enforces its ACL exactly as it
		// did when this package owned the dispatch.
		spec := &mcpsdk.Tool{Name: scope, InputSchema: &jsonschema.Schema{Type: "object"}}
		if handler, ok := handlers[scope]; ok {
			srv.AddTool(spec, builtinToolAdapter(env, handler))
			return
		}
		srv.AddTool(spec, dynamicToolHandler(env, scope))
		return
	}

	specs := builtinTools(ctx, env)
	registered := make(map[string]bool, len(specs))
	for _, spec := range specs {
		registered[spec.Name] = true
	}

	// Add metadata for the callable-but-unlisted tools so they can be invoked.
	for _, spec := range append(adminGraphQLTools(), federatedGraphQLTool(defaultKBIDNote)) {
		if !registered[spec.Name] {
			registered[spec.Name] = true
			specs = append(specs, spec)
		}
	}

	for _, spec := range specs {
		handler, ok := handlers[spec.Name]
		if !ok {
			// No built-in implementation: a note-registered (mcp_method) tool,
			// whose response is the note body.
			srv.AddTool(sdkTool(spec), dynamicToolHandler(env, spec.Name))
			continue
		}
		srv.AddTool(sdkTool(spec), builtinToolAdapter(env, handler))
	}
}

// sdkTool converts our tool metadata into the SDK's shape, keeping the
// agent-facing descriptions and schemas byte-identical.
func sdkTool(t Tool) *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: inputSchemaToJSONSchema(t.InputSchema),
	}
}

func inputSchemaToJSONSchema(in *InputSchema) *jsonschema.Schema {
	// The SDK requires an object schema on every tool; a tool declaring no
	// arguments still needs the empty object.
	if in == nil {
		return &jsonschema.Schema{Type: "object"}
	}
	out := &jsonschema.Schema{Type: in.Type, Required: in.Required}
	if len(in.Properties) > 0 {
		out.Properties = make(map[string]*jsonschema.Schema, len(in.Properties))
		for name, prop := range in.Properties {
			out.Properties[name] = propertyToJSONSchema(prop)
		}
	}
	return out
}

func propertyToJSONSchema(p Property) *jsonschema.Schema {
	s := &jsonschema.Schema{Type: p.Type, Description: p.Description}
	if p.Items != nil {
		s.Items = propertyToJSONSchema(*p.Items)
	}
	return s
}

// builtinToolAdapter bridges a handler that speaks our JSON-RPC Response into
// an SDK tool handler. A Response carrying an Error becomes a *jsonrpc.Error,
// which the SDK puts on the wire as a real JSON-RPC error rather than an
// isError result — preserving the error codes and messages clients already
// depend on.
func builtinToolAdapter(env Env, handler toolHandler) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return toSDKResult(handler(ctx, env, nil, rawArguments(req)))
	}
}

func dynamicToolHandler(env Env, name string) mcpsdk.ToolHandler {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return toSDKResult(handleDynamicMethod(ctx, env, nil, name))
	}
}

// noopToolHandler is never invoked: it backs the name-only registrations that
// exist purely so initialize advertises the tools capability.
func noopToolHandler(context.Context, *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	return nil, &jsonrpc.Error{Code: ErrCodeInternal, Message: "Internal Error"}
}

// rawArguments returns the tool arguments as raw JSON, exactly as the client
// sent them, so the handlers keep doing their own lenient decoding.
func rawArguments(req *mcpsdk.CallToolRequest) json.RawMessage {
	if req == nil || req.Params == nil {
		return nil
	}
	return req.Params.Arguments
}

func toSDKResult(resp Response) (*mcpsdk.CallToolResult, error) {
	if resp.Error != nil {
		return nil, &jsonrpc.Error{Code: int64(resp.Error.Code), Message: resp.Error.Message}
	}

	result, ok := resp.Result.(CallToolResult)
	if !ok {
		return nil, &jsonrpc.Error{Code: ErrCodeInternal, Message: "Internal Error"}
	}

	content := make([]mcpsdk.Content, 0, len(result.Content))
	for _, item := range result.Content {
		content = append(content, &mcpsdk.TextContent{Text: item.Text})
	}
	return &mcpsdk.CallToolResult{
		Content:           content,
		StructuredContent: result.StructuredContent,
		IsError:           result.IsError,
	}, nil
}

// listedToolsMiddleware narrows tools/list to the tools this caller may see.
// Registration covers everything callable, so without this filter a federation
// peer would start seeing graphql_request advertised.
func listedToolsMiddleware(buildCtx context.Context, env Env) mcpsdk.Middleware {
	return func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(ctx context.Context, method string, req mcpsdk.Request) (mcpsdk.Result, error) {
			res, err := next(ctx, method, req)
			if method != mcpMethodToolsList || err != nil {
				return res, err
			}
			listing, ok := res.(*mcpsdk.ListToolsResult)
			if !ok {
				return res, err
			}

			visible := make(map[string]bool)
			for _, t := range builtinTools(buildCtx, env) {
				visible[t.Name] = true
			}
			kept := make([]*mcpsdk.Tool, 0, len(listing.Tools))
			for _, t := range listing.Tools {
				if visible[t.Name] {
					kept = append(kept, t)
				}
			}
			listing.Tools = kept
			return listing, nil
		}
	}
}

// unknownToolMiddleware restores the endpoint's long-standing answer for a tool
// nobody registered: -32601 "Method not found: <name>". The SDK reports an
// unknown tool as -32602 instead, and clients (plus the note-tool ACL path,
// where an unreadable mcp_method note is simply never registered) rely on the
// method-not-found shape.
func unknownToolMiddleware() mcpsdk.Middleware {
	return func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(ctx context.Context, method string, req mcpsdk.Request) (mcpsdk.Result, error) {
			res, err := next(ctx, method, req)
			if method != mcpMethodToolsCall || err == nil {
				return res, err
			}
			var wireErr *jsonrpc.Error
			if !errors.As(err, &wireErr) || !strings.HasPrefix(wireErr.Message, "unknown tool ") {
				return res, err
			}
			call, ok := req.(*mcpsdk.CallToolRequest)
			if !ok {
				return res, err
			}
			return nil, &jsonrpc.Error{
				Code:    ErrCodeMethodNotFound,
				Message: "Method not found: " + call.Params.Name,
			}
		}
	}
}
