// Package codellmgql is codellm's own GraphQL surface: markdown STRUCTURE only
// (editor/debugger support), separate from the monolith's GraphQL and from
// fleet's fleetgql. It exposes parseMarkdown / assembleMarkdown — the block
// boundaries come from codellm's OWN ExtractFencedBlocks so they match execution
// exactly. Execution itself stays the OpenAI REST path. See
// docs/dev/fleet_graphql.md.
package codellmgql

import (
	"context"
)

//go:generate go tool github.com/99designs/gqlgen generate --config gqlgen.yml

// BlockRunRequest contains a markdown execution request.
type BlockRunRequest struct {
	Body       string
	FleetInput []byte
	MaxSteps   int
}

type BlockResult struct {
	Index    int
	ExitCode int
	Stdout   string
	Stderr   string
	Pipe     string
}

type BlockRunResult struct {
	Output  string
	Results []BlockResult
}

type BlockRunner interface {
	RunBlocks(context.Context, BlockRunRequest) (BlockRunResult, error)
}

// Resolver is the gqlgen root resolver. parseMarkdown / assembleMarkdown are
// pure functions over the request input (no state, no dependencies).
type Resolver struct{ runner BlockRunner }

// NewResolver builds a Resolver.
func NewResolver(runner BlockRunner) *Resolver {
	return &Resolver{runner: runner}
}
