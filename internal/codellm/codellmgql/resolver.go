// Package codellmgql is codellm's own GraphQL surface: markdown STRUCTURE only
// (editor/debugger support), separate from the monolith's GraphQL and from
// fleet's fleetgql. It exposes parseMarkdown / assembleMarkdown — the block
// boundaries come from codellm's OWN ExtractFencedBlocks so they match execution
// exactly. Execution itself stays the OpenAI REST path. See
// docs/dev/fleet_graphql.md.
package codellmgql

//go:generate go tool github.com/99designs/gqlgen generate --config gqlgen.yml

// Resolver is the gqlgen root resolver. parseMarkdown / assembleMarkdown are
// pure functions over the request input (no state, no dependencies).
type Resolver struct{}

// NewResolver builds a Resolver.
func NewResolver() *Resolver { return &Resolver{} }
