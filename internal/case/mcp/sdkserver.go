package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverVersion = "1.0.0"

// buildServer assembles the per-request MCP server. It is per-request because
// both the tool set and the instructions depend on who is calling: an admin API
// key sees the GraphQL tools, a federation peer gets a scoped graphql_request,
// and dynamic note tools are filtered by the caller's read access.
// scope names the single tool a request needs registered. A tools/call only
// dispatches one tool, so registering the whole catalog — which means walking
// every note and access-checking each mcp_method one — would put work on the
// hot path that the endpoint never used to do. An empty scope is tools/list,
// which is answered by a shared listing server (see listingServer).
func buildServer(ctx context.Context, env Env, initResult initialization, scope string) *mcpsdk.Server {
	if scope == "" {
		return listingServer(builtinTools(ctx, env), initResult)
	}

	srv := newServer(initResult)
	registerTools(srv, env, scope)
	srv.AddReceivingMiddleware(unknownToolMiddleware())

	return srv
}

func newServer(initResult initialization) *mcpsdk.Server {
	return mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: initResult.ServerName, Title: initResult.ServerName, Version: serverVersion},
		&mcpsdk.ServerOptions{Instructions: initResult.Instructions, Capabilities: serverCapabilities()},
	)
}

// serverCapabilities advertises tools and nothing else. No listChanged: every
// request gets a fresh, stateless server, so a list_changed notification has no
// session to go to; a client that sees none opens no subscriptions/listen and
// reads the list again when it reconnects, which is all an mcp_method note edit
// needs. No logging: the server sends no log messages.
func serverCapabilities() *mcpsdk.ServerCapabilities {
	return &mcpsdk.ServerCapabilities{Tools: &mcpsdk.ToolCapabilities{}}
}

// registerTools registers the one tool a tools/call dispatches, or for any
// other method the bare names initialize advertises the tools capability with.
func registerTools(srv *mcpsdk.Server, env Env, scope string) {
	handlers := builtinToolHandlers()

	if scope == scopeCapabilities {
		for name := range handlers {
			srv.AddTool(&mcpsdk.Tool{Name: name, InputSchema: &jsonschema.Schema{Type: "object"}}, noopToolHandler)
		}
		return
	}

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
}

// listingServers keeps one tools/list server per distinct visible catalog.
// Building a server registers every tool, and the SDK re-parses each schema
// twice in AddTool, which made a static 15 KB listing cost about a millisecond
// of CPU and 1.6 MB of garbage. A listing server holds no-op handlers and no
// caller state, so callers who see the same catalog can share it. The catalogs
// are few (anonymous, API key, admin key, per-ACL note tools); the map is
// cleared if note edits ever accumulate past maxListingServers of them.
var listingServers = struct { //nolint:gochecknoglobals // process-wide cache of caller-free listing servers
	sync.Mutex
	byCatalog map[string]*mcpsdk.Server
}{byCatalog: map[string]*mcpsdk.Server{}}

const maxListingServers = 256

func listingServer(specs []Tool, initResult initialization) *mcpsdk.Server {
	catalog := struct {
		Tools []Tool         `json:"tools"`
		Init  initialization `json:"init"`
	}{specs, initResult}
	key, err := json.Marshal(catalog)
	if err != nil {
		return buildListingServer(specs, initResult)
	}

	listingServers.Lock()
	defer listingServers.Unlock()

	if srv, ok := listingServers.byCatalog[string(key)]; ok {
		return srv
	}
	if len(listingServers.byCatalog) >= maxListingServers {
		listingServers.byCatalog = map[string]*mcpsdk.Server{}
	}
	srv := buildListingServer(specs, initResult)
	listingServers.byCatalog[string(key)] = srv
	return srv
}

func buildListingServer(specs []Tool, initResult initialization) *mcpsdk.Server {
	srv := newServer(initResult)
	for _, spec := range specs {
		srv.AddTool(sdkTool(spec), noopToolHandler)
	}
	return srv
}

// sdkTool converts our tool metadata into the SDK's shape, keeping the
// agent-facing descriptions and schemas byte-identical.
func sdkTool(t Tool) *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: marshaledInputSchema(t.InputSchema),
	}
}

// renderedSchemas holds each distinct input schema as the bytes jsonschema
// renders for it. Every tools/list registers the whole catalog on a fresh
// server, and jsonschema's MarshalJSON, run once in AddTool and again for the
// response, was most of that request's CPU. The schemas are a small, fixed set
// (the compiled-in tools plus the notes' mcp_method ones), so the map stays
// small.
var renderedSchemas sync.Map //nolint:gochecknoglobals // process-wide cache of rendered schemas

func marshaledInputSchema(in *InputSchema) json.RawMessage {
	key, err := json.Marshal(in)
	if err != nil {
		return mustMarshalSchema(inputSchemaToJSONSchema(in))
	}
	if cached, ok := renderedSchemas.Load(string(key)); ok {
		rendered, isRaw := cached.(json.RawMessage)
		if isRaw {
			return rendered
		}
	}
	rendered := mustMarshalSchema(inputSchemaToJSONSchema(in))
	renderedSchemas.Store(string(key), rendered)
	return rendered
}

func mustMarshalSchema(s *jsonschema.Schema) json.RawMessage {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
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
// exist purely so initialize advertises the tools capability, and the shared
// tools/list servers, which only ever list.
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
