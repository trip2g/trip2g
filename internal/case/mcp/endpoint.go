package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)

	// Parse JSON-RPC request
	var rpcReq Request
	err := json.Unmarshal(req.Req.PostBody(), &rpcReq)
	if err != nil {
		resp := errorResponse(nil, ErrCodeParseError, "Parse error: "+err.Error())
		return writeJSONResponse(req, resp)
	}

	// Validate JSON-RPC version
	if rpcReq.JSONRPC != "2.0" {
		resp := errorResponse(rpcReq.ID, ErrCodeInvalidRequest, "Invalid JSON-RPC version")
		return writeJSONResponse(req, resp)
	}

	// Enforce federation hop depth limit.
	resolveCtx := context.Context(req.Req)
	depthHeader := req.Req.Request.Header.Peek("X-MCP-Federation-Depth")
	if len(depthHeader) > 0 {
		incomingDepth, _ := strconv.Atoi(string(depthHeader))
		if incomingDepth >= env.FederationMaxDepth() {
			resp := errorResponse(rpcReq.ID, ErrCodeInternal, "federation max depth exceeded")
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
		return writeJSONResponse(req, resp)
	}

	if userToken == nil {
		newCtx, errResp := authenticateAnonymousRequest(resolveCtx, req, env, rpcReq.ID)
		if errResp != nil {
			return writeJSONResponse(req, *errResp)
		}
		resolveCtx = newCtx
	}

	// Handle request
	rpcReq.MethodOverride = string(req.Req.Request.URI().QueryArgs().Peek("method"))
	resp := Resolve(resolveCtx, env, rpcReq)
	return writeJSONResponse(req, resp)
}

// authenticateAnonymousRequest resolves API-key or federation-JWT auth when no
// personal token is present. It returns the augmented context, or an error
// response to send back unchanged when auth fails.
func authenticateAnonymousRequest(ctx context.Context, req *appreq.Request, env Env, id any) (context.Context, *Response) {
	apiKeyValue := strings.TrimSpace(string(req.Req.Request.Header.Peek("X-API-Key")))
	if apiKeyValue != "" {
		apiKey, keyErr := env.ResolveAPIKey(req.Req, apiKeyValue, "mcp")
		if keyErr != nil {
			resp := errorResponse(id, ErrCodeInternal, "Auth failed: "+keyErr.Error())
			return ctx, &resp
		}
		adminTools := apiKey.EnableMcpAdminTools != nil && *apiKey.EnableMcpAdminTools
		// Attribute internal admin GraphQL calls (WithAdminToken) to the API
		// key owner so mutations like createAdmin record a real granted_by.
		req.AdminActorUserID = int(apiKey.CreatedBy)
		return contextWithMCPAPIKeyAuth(ctx, adminTools), nil
	}

	authHeader := strings.TrimSpace(string(req.Req.Request.Header.Peek("Authorization")))
	token, isBearerToken := strings.CutPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)
	if authHeader != "" && (!isBearerToken || token == "") {
		resp := errorResponse(id, ErrCodeInternal, "Federation auth failed: malformed bearer token")
		return ctx, &resp
	}
	if !isBearerToken || token == "" {
		return ctx, nil
	}
	kid, allowedSubgraphs, verifyErr := verifyInbound(req.Req, env, token)
	if verifyErr != nil {
		resp := errorResponse(id, ErrCodeInternal, "Federation auth failed: "+verifyErr.Error())
		return ctx, &resp
	}
	return contextWithFederationAuth(ctx, kid, allowedSubgraphs), nil
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
	return "/_system/mcp"
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

Send POST requests with a JSON-RPC 2.0 body.

Public access: no token required for open knowledge bases.

Authentication (required for private/subscriber-only content):
  Authorization: Bearer t2g_<your-token>
  ?token=t2g_<your-token>
  X-API-Key: <your-api-key>

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

Extend MCP via note frontmatter:
  mcp_method: <name>    — expose a note as an MCP tool; the note's content becomes
                          the tool's response.
  mcp_method: initialize — default method shown when an agent requests tool info.
                           Override which note handles it via ?method=<note-path>.
Read more in the docs.

Docs: https://trip2g.com/en/user/mcp
`, "{{MCP_URL}}", mcpURL)

	req.Req.SetContentType("text/plain; charset=utf-8")
	req.Req.SetStatusCode(http.StatusOK)
	req.Req.SetBodyString(body)
	return nil, nil
}

func (*GetEndpoint) Path() string {
	return "/_system/mcp"
}

func (*GetEndpoint) Method() string {
	return http.MethodGet
}
