// Package trip2ggql holds the genqlient-generated, type-safe GraphQL client for
// the fleet's trip2g admin-lane operations. The operations live in
// operations.graphql and are validated against the trip2g schema at code-gen
// time, so a wrong field or nesting fails `go generate` instead of at runtime.
//
//go:generate go tool github.com/Khan/genqlient
package trip2ggql
