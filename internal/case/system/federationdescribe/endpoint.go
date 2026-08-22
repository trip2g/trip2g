// Package federationdescribe answers a peer that asks about the pairing it
// holds, rather than about content.
//
// A sibling of /_system/mcp rather than a tool on it, because the question is a
// different kind: not "what do you know" but "what is this pairing". That grows
// by adding fields — capabilities, protocol version, whatever comes next — where
// a tool would grow by adding methods, and each of those would be another
// unlisted name on a surface whose whole contract is that it has six.
package federationdescribe

import (
	"net/http"
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
)

type Endpoint struct{}

func (*Endpoint) Path() string   { return "/_system/mcp/federation" }
func (*Endpoint) Method() string { return http.MethodGet }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)

	// Signed like every other federated call, so the answer is about the pairing
	// that asked and can never be about another. A GET carries no body, so no
	// body digest is expected.
	header := strings.TrimSpace(string(req.Req.Request.Header.Peek("Authorization")))
	token, isBearer := strings.CutPrefix(header, "Bearer ")
	token = strings.TrimSpace(token)
	if !isBearer || token == "" {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		return nil, nil
	}

	kid, err := mcp.VerifyFederationBearer(req.Req, env, token, nil)
	if err != nil {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		//nolint:nilerr // intentional: a caller who cannot prove a pairing gets 401 and no detail, not a 500
		return nil, nil
	}

	return Resolve(req.Req, env, kid)
}
