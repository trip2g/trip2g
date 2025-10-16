package applygitchanges

import (
	"context"
	"fmt"
)

type Env interface {
	ApplyGitChanges(ctx context.Context) ([]string, error)
}

type Result struct {
	ChangedFiles []string
}

func Resolve(ctx context.Context, env Env) (*Result, error) {
	changedFiles, err := env.ApplyGitChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to apply git changes: %w", err)
	}

	res := Result{
		ChangedFiles: changedFiles,
	}

	return &res, nil
}
