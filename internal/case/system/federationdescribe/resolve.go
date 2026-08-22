package federationdescribe

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./resolve.go

import (
	"context"

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

func Resolve(ctx context.Context, env Env, kid string) (*Description, error) {
	scope, err := env.ListFederationSecretScopeByKID(ctx, kid)
	if err != nil {
		return nil, err
	}

	entries := make([]ScopeEntry, 0, len(scope))
	for _, row := range scope {
		entries = append(entries, ScopeEntry{Name: row.Name, HumanDescription: row.HumanDescription})
	}

	return &Description{
		Version:   Version,
		Kid:       kid,
		Subgraphs: entries,
		Rotation:  true,
	}, nil
}
