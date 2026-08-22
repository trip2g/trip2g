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
	"context"
	"net/http"
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
)

// Version is what the shape below is. A reader meeting one it does not know can
// say so instead of reading fields that have moved.
const Version = 1

type Env interface {
	mcp.FederationVerifyEnv
	ListFederationSecretScopeByKID(ctx context.Context, kid string) ([]db.ListFederationSecretScopeByKIDRow, error)
}

// Description is the whole answer, and only what the caller is entitled to: the
// pairing it already authenticated as, and what that pairing may see. Nothing
// about this instance's other peers, nothing about subgraphs outside the grant.
type Description struct {
	Version int    `json:"version"`
	Kid     string `json:"kid"`
	// Subgraphs is what this kid may surface here. Empty is a real answer, not an
	// error: a pairing scoped to nothing authenticates and returns nothing, and
	// saying so is the only way the asking side tells that apart from a query
	// that matched nothing.
	Subgraphs []ScopeEntry `json:"subgraphs"`
	// Rotation says this instance can replace the pairing's key, so a peer learns
	// it before trying rather than from a refusal.
	Rotation bool `json:"rotation"`
}

type ScopeEntry struct {
	Name string `json:"name"`
	// HumanDescription is what the subgraph is, as its owner wrote it. A slug
	// tells a peer nothing, and only the granting side knows the meaning.
	HumanDescription string `json:"human_description"`
}

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

func Resolve(ctx context.Context, env Env, kid string) (Description, error) {
	scope, err := env.ListFederationSecretScopeByKID(ctx, kid)
	if err != nil {
		return Description{}, err
	}

	entries := make([]ScopeEntry, 0, len(scope))
	for _, row := range scope {
		entries = append(entries, ScopeEntry{Name: row.Name, HumanDescription: row.HumanDescription})
	}

	return Description{
		Version:   Version,
		Kid:       kid,
		Subgraphs: entries,
		Rotation:  true,
	}, nil
}
