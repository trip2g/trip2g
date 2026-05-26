package listsecretkeys

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg listsecretkeys_test . Env

import (
	"context"
	"fmt"
)

type Env interface {
	ListSecretKeys(ctx context.Context, like string) ([]string, error)
}

func Resolve(ctx context.Context, env Env, like string) ([]string, error) {
	if like == "" {
		like = "%"
	}

	keys, err := env.ListSecretKeys(ctx, like)
	if err != nil {
		return nil, fmt.Errorf("failed to list secret keys: %w", err)
	}

	return keys, nil
}
