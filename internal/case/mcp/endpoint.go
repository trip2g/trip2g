package mcp

import (
	"context"
	"encoding/json"
	"net/http"
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

	resolveCtx := context.Context(req.Req)
	if authHeader := strings.TrimSpace(string(req.Req.Request.Header.Peek("Authorization"))); authHeader != "" {
		token, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			resp := errorResponse(rpcReq.ID, ErrCodeInternal, "Federation auth failed: malformed bearer token")
			return writeJSONResponse(req, resp)
		}

		kid, allowedSubgraphs, verifyErr := verifyInbound(req.Req, env, strings.TrimSpace(token))
		if verifyErr != nil {
			resp := errorResponse(rpcReq.ID, ErrCodeInternal, "Federation auth failed: "+verifyErr.Error())
			return writeJSONResponse(req, resp)
		}
		resolveCtx = contextWithFederationAuth(resolveCtx, kid, allowedSubgraphs)
	}

	// Handle request
	resp := Resolve(resolveCtx, env, rpcReq)
	return writeJSONResponse(req, resp)
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
