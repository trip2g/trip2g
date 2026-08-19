package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trip2g/internal/appreq"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

const (
	mcpEndpointPath = "/_system/mcp"
	// streamableAcceptHeader is what the MCP Streamable HTTP spec requires
	// clients to send; the transport rejects a request without both types.
	streamableAcceptHeader = "application/json, text/event-stream"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	start := time.Now()

	// The body is parsed here as well as by the SDK: auth failures, the hop-depth
	// limit and the ?method= override all have to answer with the request's own
	// JSON-RPC id before the transport ever sees the message.
	var rpcReq Request
	err := json.Unmarshal(req.Req.PostBody(), &rpcReq)
	if err != nil {
		resp := errorResponse(nil, ErrCodeParseError, "Parse error: "+err.Error())
		return writeJSONResponse(req, resp)
	}

	m := env.MCPMetrics()

	if rpcReq.JSONRPC != "2.0" {
		resp := errorResponse(rpcReq.ID, ErrCodeInvalidRequest, "Invalid JSON-RPC version")
		recordRejectedRequest(m, rpcReq, authAnonymous, time.Since(start).Seconds())
		return writeJSONResponse(req, resp)
	}

	// Enforce the federation hop-depth limit as a loop-protection backstop. Depth
	// is observed for every request: no header, a malformed value or a negative one
	// all count as 0 = direct client. The comparison is inclusive (a path of N
	// segments reaches depth N and is allowed when N <= max); explicit deep paths
	// are rejected earlier and more cleanly by federationPathDepthExceeded, so this
	// counter mainly bounds fan-out cycles where peers federate back with no
	// explicit segments to count.
	resolveCtx := context.Context(req.Req)
	depthHeader := req.Req.Request.Header.Peek("X-MCP-Federation-Depth")
	incomingDepth := 0
	if len(depthHeader) > 0 {
		if v, parseErr := strconv.Atoi(string(depthHeader)); parseErr == nil && v > 0 {
			incomingDepth = v
		}
	}
	m.ObserveFederationDepth(incomingDepth)
	if len(depthHeader) > 0 {
		if incomingDepth > env.FederationMaxDepth() {
			resp := errorResponse(rpcReq.ID, ErrCodeInternal, "federation max depth exceeded")
			recordRejectedRequest(m, rpcReq, authAnonymous, time.Since(start).Seconds())
			return writeJSONResponse(req, resp)
		}
		resolveCtx = contextWithFederationDepth(resolveCtx, incomingDepth)
	}

	// Resolve personal token first (t2g_* Bearer or ?token= — handled by appreq.UserToken).
	// If a user is present in ctx, the personal token authenticated successfully; skip federation verifyInbound.
	// If UserToken returns an error (invalid/revoked/expired personal token), surface it immediately.
	// Only attempt verifyInbound when no personal token user was found AND a Bearer is present.
	userToken, utErr := req.UserToken()
	if utErr != nil {
		resp := errorResponse(rpcReq.ID, ErrCodeInternal, "Auth failed: "+utErr.Error())
		recordRejectedRequest(m, rpcReq, authToken, time.Since(start).Seconds())
		return writeJSONResponse(req, resp)
	}

	if userToken == nil {
		newCtx, authAttempt, errResp := authenticateAnonymousRequest(resolveCtx, req, env, rpcReq.ID)
		if errResp != nil {
			recordRejectedRequest(m, rpcReq, authAttempt, time.Since(start).Seconds())
			return writeJSONResponse(req, *errResp)
		}
		resolveCtx = newCtx
	}

	resolveCtx = ContextWithMetrics(resolveCtx, m)

	// ?method= selects which note supplies the initialize instructions.
	rpcReq.MethodOverride = string(req.Req.Request.URI().QueryArgs().Peek("method"))

	// The client's own request is forwarded, so the transport applies its real
	// content negotiation rather than anything reconstructed here.
	httpReq := &http.Request{}
	err = fasthttpadaptor.ConvertRequest(req.Req, httpReq, true)
	if err != nil {
		return nil, err
	}

	answer := serveJSONRPC(resolveCtx, env, rpcReq, httpReq)

	req.Req.SetStatusCode(answer.status)
	if answer.contentType != "" {
		req.Req.SetContentType(answer.contentType)
	}
	req.Req.SetBody(answer.body)

	recordRequestMetrics(resolveCtx, m, rpcReq, userToken != nil, decodeResponse(answer.body), time.Since(start).Seconds())
	return nil, nil
}

// rpcAnswer is one fully-formed HTTP response from the MCP transport.
type rpcAnswer struct {
	status      int
	contentType string
	body        []byte
}

// serveJSONRPC is the single implementation behind both entry points: Resolve,
// the use-case API, and Endpoint.Handle, the HTTP adapter over it. Everything
// between an authenticated context and a finished response lives here, so the
// two entry points cannot drift apart.
//
// httpReq carries the message to the transport. Endpoint.Handle passes the
// client's real request; Resolve synthesises a spec-compliant one.
func serveJSONRPC(ctx context.Context, env Env, rpcReq Request, httpReq *http.Request) rpcAnswer {
	// Only initialize surfaces the instructions, and resolving them scans the
	// whole corpus — so every other method skips that work, as it always has.
	instructions := ""
	if rpcReq.Method == MCPMethodInitialize {
		resolved, instrErr := initializeInstructions(ctx, env, rpcReq.MethodOverride)
		if instrErr != nil {
			return jsonAnswer(Response{JSONRPC: "2.0", ID: rpcReq.ID, Error: instrErr})
		}
		instructions = resolved
	}

	if errResp := checkToolCallParams(rpcReq); errResp != nil {
		return jsonAnswer(*errResp)
	}
	setNormalizedBody(httpReq, rpcReq)

	rec := serveMessage(ctx, env, instructions, toolScope(rpcReq), httpReq)

	if restored, ok := restoreUnhandledMethod(rec.body, rpcReq); ok {
		return jsonAnswer(restored)
	}
	if restored, ok := restoreMethodNotFound(rec.body, rpcReq); ok {
		return jsonAnswer(restored)
	}

	// An empty content type is left alone: the transport omits it on the
	// bodiless 202 it answers notifications with.
	return rpcAnswer{status: rec.status, contentType: rec.header.Get("Content-Type"), body: rec.body}
}

// jsonAnswer renders a JSON-RPC response this package produced itself, rather
// than one that came back from the transport.
func jsonAnswer(resp Response) rpcAnswer {
	body, err := json.Marshal(resp)
	if err != nil {
		return rpcAnswer{status: http.StatusInternalServerError, contentType: "application/json"}
	}
	return rpcAnswer{status: http.StatusOK, contentType: "application/json", body: body}
}

// serveMessage runs one HTTP request through the official Streamable HTTP
// transport against a server built for this caller.
func serveMessage(ctx context.Context, env Env, instructions, scope string, httpReq *http.Request) *responseRecorder {
	handler := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return buildServer(ctx, env, instructions, scope) },
		&mcpsdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)

	rec := &responseRecorder{header: http.Header{}, status: http.StatusOK}
	handler.ServeHTTP(rec, httpReq.WithContext(ctx))
	return rec
}

// responseRecorder is the minimal http.ResponseWriter needed to hand the SDK's
// net/http response back to fasthttp.
type responseRecorder struct {
	header http.Header
	body   []byte
	status int
	wrote  bool
}

func (r *responseRecorder) Header() http.Header { return r.header }

func (r *responseRecorder) Write(p []byte) (int, error) {
	r.body = append(r.body, p...)
	return len(p), nil
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wrote {
		return
	}
	r.wrote = true
	r.status = status
}

// setNormalizedBody substitutes an empty params object on initialize. The SDK
// rejects a params-less initialize with an HTTP 400, but this endpoint has
// always accepted it bare — `{"jsonrpc":"2.0","id":1,"method":"initialize"}` is
// what the ?method= docs tell agents to probe with — so the answer stays a
// JSON-RPC response.
func setNormalizedBody(httpReq *http.Request, rpcReq Request) {
	if len(rpcReq.Params) > 0 || rpcReq.Method != MCPMethodInitialize {
		return
	}
	rpcReq.Params = json.RawMessage(`{}`)
	body, err := json.Marshal(rpcReq)
	if err != nil {
		return
	}
	httpReq.Body = io.NopCloser(bytes.NewReader(body))
	httpReq.ContentLength = int64(len(body))
}

// scopeCapabilities registers tool names only. initialize has to advertise the
// tools capability — the SDK omits it entirely when nothing is registered — but
// it never surfaces tool descriptions or schemas, so the descriptive catalog is
// not built for it.
const scopeCapabilities = "\x00capabilities"

// toolScope names the only tool a request can dispatch, so the server is built
// with just that one registered. tools/list needs the full catalog and gets an
// empty scope.
func toolScope(rpcReq Request) string {
	if rpcReq.Method == mcpMethodToolsList {
		return ""
	}
	if rpcReq.Method != mcpMethodToolsCall {
		return scopeCapabilities
	}
	var params CallToolParams
	if json.Unmarshal(rpcReq.Params, &params) != nil {
		return ""
	}
	return params.Name
}

// checkToolCallParams reproduces the params validation tools/call did when this
// handler owned the dispatch: without a decodable params object the caller gets
// invalid-params, not the transport's unknown-tool answer.
func checkToolCallParams(rpcReq Request) *Response {
	if rpcReq.Method != mcpMethodToolsCall {
		return nil
	}
	var params CallToolParams
	err := json.Unmarshal(rpcReq.Params, &params)
	if err != nil {
		resp := errorResponse(rpcReq.ID, ErrCodeInvalidParams, "Invalid params: "+err.Error())
		return &resp
	}
	return nil
}

// restoreMethodNotFound undoes an SDK quirk: jsonrpc2 matches errors by code
// alone (WireError.Is ignores the message), so every -32601 a tool handler
// returns comes back out as the generic `method not found: "tools/call"`.
// Clients — and the note-tool ACL path, where an unreadable mcp_method note is
// simply never registered — depend on the name being the tool's, not the
// transport method's.
func restoreMethodNotFound(body []byte, rpcReq Request) (Response, bool) {
	var resp Response
	if json.Unmarshal(body, &resp) != nil || resp.Error == nil {
		return Response{}, false
	}

	restored := methodNotFoundMessage(resp.Error, rpcReq)
	if restored == resp.Error.Message {
		return Response{}, false
	}

	resp.Error.Message = restored
	return resp, true
}

// restoreUnhandledMethod turns the transport's plain-text 400 for a method it
// does not implement back into a JSON-RPC error at 200, the way this endpoint
// has always answered. A JSON-RPC caller gets a JSON-RPC answer; which methods
// exist is still decided entirely by the SDK, so anything it grows support for
// keeps working without changes here.
func restoreUnhandledMethod(body []byte, rpcReq Request) (Response, bool) {
	if !bytes.HasPrefix(body, []byte("JSON RPC not handled:")) {
		return Response{}, false
	}
	return errorResponse(rpcReq.ID, ErrCodeMethodNotFound, "Method not found: "+rpcReq.Method), true
}

// methodNotFoundMessage returns the message a -32601 should carry: the tool's
// own name for a tools/call, the JSON-RPC method otherwise. Anything that is
// not the SDK's generic rewrite is returned unchanged.
func methodNotFoundMessage(rpcErr *Error, rpcReq Request) string {
	if rpcErr.Code != ErrCodeMethodNotFound || !strings.HasPrefix(rpcErr.Message, "method not found: ") {
		return rpcErr.Message
	}

	name := rpcReq.Method
	if rpcReq.Method == mcpMethodToolsCall {
		var params CallToolParams
		if json.Unmarshal(rpcReq.Params, &params) != nil || params.Name == "" {
			return rpcErr.Message
		}
		name = params.Name
	}
	return "Method not found: " + name
}

// decodeResponse re-reads the transport's own output so request metrics keep
// classifying results the way they did when this handler owned the dispatch.
func decodeResponse(body []byte) Response {
	var resp Response
	if json.Unmarshal(body, &resp) != nil {
		return Response{}
	}
	return resp
}

// authenticateAnonymousRequest resolves API-key or federation-JWT auth when no
// personal token is present. It returns the augmented context, or an error
// response to send back unchanged when auth fails, plus the attempted auth
// kind for metric labels (an invalid API key is still auth=api_key traffic).
func authenticateAnonymousRequest(ctx context.Context, req *appreq.Request, env Env, id any) (context.Context, string, *Response) {
	apiKeyValue := strings.TrimSpace(string(req.Req.Request.Header.Peek("X-API-Key")))
	if apiKeyValue != "" {
		apiKey, keyErr := env.ResolveAPIKey(req.Req, apiKeyValue, "mcp")
		if keyErr != nil {
			resp := errorResponse(id, ErrCodeInternal, "Auth failed: "+keyErr.Error())
			return ctx, authAPIKey, &resp
		}
		adminTools := apiKey.EnableMcpAdminTools != nil && *apiKey.EnableMcpAdminTools
		// Attribute internal admin GraphQL calls (WithAdminToken) to the API
		// key owner so mutations like createAdmin record a real granted_by.
		req.AdminActorUserID = int(apiKey.CreatedBy)
		return contextWithMCPAPIKeyAuth(ctx, adminTools), authAPIKey, nil
	}

	authHeader := strings.TrimSpace(string(req.Req.Request.Header.Peek("Authorization")))
	token, isBearerToken := strings.CutPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)
	if authHeader != "" && (!isBearerToken || token == "") {
		resp := errorResponse(id, ErrCodeInternal, "Federation auth failed: malformed bearer token")
		return ctx, authFederation, &resp
	}
	if !isBearerToken || token == "" {
		return ctx, authAnonymous, nil
	}
	kid, allowedSubgraphs, verifyErr := verifyInbound(req.Req, env, token)
	if verifyErr != nil {
		resp := errorResponse(id, ErrCodeInternal, "Federation auth failed: "+verifyErr.Error())
		return ctx, authFederation, &resp
	}
	return contextWithFederationAuth(ctx, kid, allowedSubgraphs), authFederation, nil
}

func writeJSONResponse(req *appreq.Request, resp Response) (interface{}, error) {
	req.Req.SetContentType("application/json")
	req.Req.SetStatusCode(http.StatusOK)

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	req.Req.SetBody(data)
	return nil, nil
}

func (*Endpoint) Path() string {
	return mcpEndpointPath
}

func (*Endpoint) Method() string {
	return http.MethodPost
}

// GetEndpoint serves a human-readable info page for agents that probe /_system/mcp via GET.
type GetEndpoint struct{}

func (*GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	mcpURL := strings.TrimRight(env.PublicURL(), "/") + "/_system/mcp"

	body := strings.ReplaceAll(`This is an MCP POST endpoint (Model Context Protocol, Streamable HTTP).

Send POST requests with a JSON-RPC 2.0 body and these headers:
  Content-Type: application/json
  Accept: application/json, text/event-stream

Public access: no token required for open knowledge bases.

Authentication (required for private/subscriber-only content), any one of:
  header       Authorization: Bearer t2g_<your-token>
  query param  ?token=t2g_<your-token>
  header       X-API-Key: <your-api-key>

Get a personal token: your account → Tokens → Generate token.

Client config (Claude Desktop / Claude Code / Cursor / Copilot / Gemini CLI):

  Anonymous (public KB):
  {
    "mcpServers": {
      "trip2g": {
        "type": "http",
        "url": "{{MCP_URL}}"
      }
    }
  }

  Authenticated (private pages):
  {
    "mcpServers": {
      "trip2g": {
        "type": "http",
        "url": "{{MCP_URL}}",
        "headers": { "Authorization": "Bearer t2g_<your-token>" }
      }
    }
  }

Tools:
  search, note_html, expand, similar — search this base and read it, whole notes
    or one section at a time (toc_path), or one chunk (match_id).
  federated_search, federated_note_html, federated_expand, federated_similar,
    federated_instructions — the same against a connected peer base, addressed by
    kb_id; nested bases are addressed <hub>/<base>.

Extend MCP via note frontmatter:
  mcp_method: <name>    — expose a note as an MCP tool; the note's content becomes
                          the tool's response.
  mcp_method: initialize — the note whose content is returned in the instructions
                           field of the initialize response. This does not
                           override the JSON-RPC initialize method itself. Pick a
                           different note with ?method=<note-path>.
Read more in the docs.

Docs: https://trip2g.com/en/user/mcp
`, "{{MCP_URL}}", mcpURL)

	req.Req.SetContentType("text/plain; charset=utf-8")
	req.Req.SetStatusCode(http.StatusOK)
	req.Req.SetBodyString(body)
	return nil, nil
}

func (*GetEndpoint) Path() string {
	return mcpEndpointPath
}

func (*GetEndpoint) Method() string {
	return http.MethodGet
}
