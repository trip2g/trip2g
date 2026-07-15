package codellmgql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"

	"trip2g/internal/codellm/codellmgql/codellmgen"
)

// NewHTTPHandler builds codellm's markdown-structure GraphQL handler
// (parseMarkdown / assembleMarkdown), wrapped by auth. Introspection is enabled
// so the admin debugger tooling can explore the schema.
//
// AUTH SEAM: auth is the per-request middleware. Pass nil for the no-op
// pass-through; the caller (internal/codellm.Server) wraps this handler with the
// delegated-admin gate at the mux level, so this package builds no auth of its
// own — mirroring internal/fleet/fleetgql.NewHTTPHandler.
func NewHTTPHandler(auth func(http.Handler) http.Handler, runner BlockRunner) http.Handler {
	srv := handler.New(codellmgen.NewExecutableSchema(codellmgen.Config{Resolvers: NewResolver(runner)}))
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
