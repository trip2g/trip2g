package fleetgql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"

	"trip2g/internal/fleet/fleetgql/fleetgen"
)

// NewHTTPHandler builds the fleet GraphQL HTTP handler (POST /graphql) over the
// given role source, wrapped by auth. Introspection is enabled so admin tooling
// can explore the schema.
//
// AUTH SEAM: auth is the per-request auth middleware. Pass nil for the no-op
// pass-through — the current default until the shared delegated-admin check
// (forward the caller's cookie to the monolith's `query { viewer { role } }`;
// admin -> serve, else 401, monolith-unreachable -> fail-closed) is delivered
// as internal/delegatedadmin in a separate PR. The parameter is deliberately
// the standard func(http.Handler) http.Handler so that middleware drops in
// without touching this package — this package builds no auth of its own.
func NewHTTPHandler(roles RoleSource, auth func(http.Handler) http.Handler) http.Handler {
	srv := handler.New(fleetgen.NewExecutableSchema(fleetgen.Config{Resolvers: NewResolver(roles)}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](100))
	srv.Use(extension.Introspection{})

	var h http.Handler = srv
	if auth != nil {
		h = auth(h)
	}
	return h
}
