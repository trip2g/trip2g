package materializegitmirror

import (
	"context"
	"fmt"
)

type Env interface {
	MaterializeGitMirror(ctx context.Context) error
}

type Result struct {
	Success bool
}

func Resolve(ctx context.Context, env Env) (*Result, error) {
	if err := env.MaterializeGitMirror(ctx); err != nil {
		return nil, fmt.Errorf("failed to materialize git mirror: %w", err)
	}

	return &Result{Success: true}, nil
}
