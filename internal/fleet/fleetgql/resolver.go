// Package fleetgql is the fleet binary's own GraphQL read API: the parsed roles
// and the derived agent-interaction graph (roleGraph). It is separate from the
// monolith's GraphQL — the monolith stays a dumb note store and knows nothing
// about roles; fleet serves what only fleet understands. See
// docs/dev/fleet_graphql.md.
package fleetgql

import (
	"context"

	"trip2g/internal/fleet"
)

//go:generate go tool github.com/99designs/gqlgen generate --config gqlgen.yml

// RoleSource supplies the parsed roles the API projects. *fleet.Discovery
// satisfies it (it parses every role note under the agents-folder via
// fleet.ParseRole — the single source of truth, never reimplemented here).
type RoleSource interface {
	DiscoverParsed(ctx context.Context) ([]fleet.Role, []error)
}

// Resolver is the gqlgen root resolver. It holds only the role source; both
// queries derive their answer from the freshly-discovered roles (no caching —
// this is low-frequency admin/debug traffic).
type Resolver struct {
	roles RoleSource
}

// NewResolver builds a Resolver over a role source.
func NewResolver(roles RoleSource) *Resolver {
	return &Resolver{roles: roles}
}
