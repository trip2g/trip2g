package mcp

import (
	"encoding/json"
	"net/http"

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

	// Handle request
	resp := Resolve(req.Req, env, rpcReq)
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
