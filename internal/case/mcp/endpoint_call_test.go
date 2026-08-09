package mcp_test

import (
	"encoding/json"
	"strings"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// acceptHeader is what the MCP Streamable HTTP spec requires clients to send.
const acceptHeader = "application/json, text/event-stream"

// testTokenManager uses a cookie name no test request carries, so token
// extraction always misses and requests run as an anonymous client unless a
// case sets an auth header explicitly.
var testTokenManager = usertoken.NewManager(usertoken.Config{ //nolint:gochecknoglobals // test package global
	CookieName: "__test_cookie__",
	Secret:     "test-secret-32-bytes-long-filler!",
})

// newMCPRequest wires an appreq.Request around a fasthttp context.
func newMCPRequest(fasthttpCtx *fasthttp.RequestCtx, env interface{}) *appreq.Request {
	req := appreq.Acquire()
	req.Env = env
	req.Req = fasthttpCtx
	req.TokenManager = testTokenManager
	req.StoreInContext()
	return req
}

// callMCP runs a JSON-RPC request through the real MCP endpoint and decodes the
// answer.
//
// Tests go through Endpoint.Handle rather than any test-only entry point, so
// they exercise the one code path production uses — transport, auth, metrics
// and all — and cannot keep passing while that path breaks.
func callMCP(t *testing.T, env mcp.Env, rpcReq mcp.Request) mcp.Response {
	t.Helper()

	body, err := rawMCPCall(env, rpcReq)
	require.NoError(t, err)

	return decodeMCPResponse(t, rpcReq, body)
}

// rawMCPCall runs the request and hands back the raw response bytes without
// asserting anything, so a caller that must not fail inline — a goroutine,
// where testify's FailNow is unsafe — can collect results and check them once
// the goroutines have joined.
func rawMCPCall(env mcp.Env, rpcReq mcp.Request) ([]byte, error) {
	fasthttpCtx := &fasthttp.RequestCtx{}
	// Wire the fake server so ctx works as a context.Context (Done() panics on a bare RequestCtx).
	fasthttpCtx.Init2(nil, nil, true)
	fasthttpCtx.Request.Header.SetMethod("POST")
	fasthttpCtx.Request.SetRequestURI("/_system/mcp")
	fasthttpCtx.Request.Header.SetContentType("application/json")
	fasthttpCtx.Request.Header.Set("Accept", acceptHeader)
	fasthttpCtx.Request.SetBody(marshalMCPRequest(rpcReq))

	req := newMCPRequest(fasthttpCtx, env)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), fasthttpCtx.Response.Body()...), nil
}

// marshalMCPRequest assembles the request body by hand rather than marshaling
// the struct: Params is spliced in verbatim so a case can send deliberately
// malformed params, and a request with no id stays a true notification instead
// of gaining an "id":null the transport rejects.
func marshalMCPRequest(rpcReq mcp.Request) []byte {
	fields := []string{
		`"jsonrpc":` + quoteJSON(rpcReq.JSONRPC),
		`"method":` + quoteJSON(rpcReq.Method),
	}
	if rpcReq.ID != nil {
		id, err := json.Marshal(rpcReq.ID)
		if err == nil {
			fields = append(fields, `"id":`+string(id))
		}
	}
	if len(rpcReq.Params) > 0 {
		fields = append(fields, `"params":`+string(rpcReq.Params))
	}
	return []byte("{" + strings.Join(fields, ",") + "}")
}

func quoteJSON(s string) string {
	quoted, _ := json.Marshal(s)
	return string(quoted)
}

// decodeMCPResponse turns the wire bytes back into the typed shapes assertions
// expect. A notification gets no body at all, so the zero Response stands in.
func decodeMCPResponse(t *testing.T, rpcReq mcp.Request, body []byte) mcp.Response {
	t.Helper()

	if len(body) == 0 {
		return mcp.Response{}
	}

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(body, &resp), "response body: %s", body)

	// Decoding turns every id into a float64; JSON-RPC echoes the request's id,
	// so restore the caller's own value and keep assertions on it typed.
	if resp.ID != nil {
		resp.ID = rpcReq.ID
	}
	if resp.Error != nil {
		return resp
	}
	resp.Result = typedMCPResult(t, rpcReq.Method, body)
	return resp
}

// typedMCPResult re-decodes the result into the concrete type each method
// returns, so assertions can type-assert instead of digging through
// map[string]any.
func typedMCPResult(t *testing.T, method string, body []byte) any {
	t.Helper()

	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	if len(envelope.Result) == 0 {
		return nil
	}

	switch method {
	case "tools/list":
		var result mcp.ListToolsResult
		require.NoError(t, json.Unmarshal(envelope.Result, &result))
		return result
	case "tools/call":
		var result mcp.CallToolResult
		require.NoError(t, json.Unmarshal(envelope.Result, &result))
		// StructuredContent stays raw so callers decode it into whichever
		// payload type the tool documents — see decodePayload.
		var raw struct {
			StructuredContent json.RawMessage `json:"structuredContent"`
		}
		require.NoError(t, json.Unmarshal(envelope.Result, &raw))
		if len(raw.StructuredContent) > 0 {
			result.StructuredContent = raw.StructuredContent
		}
		return result
	default:
		var result map[string]any
		require.NoError(t, json.Unmarshal(envelope.Result, &result))
		return result
	}
}
