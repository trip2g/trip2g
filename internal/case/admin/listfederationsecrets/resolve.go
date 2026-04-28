package listfederationsecrets

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg listfederationsecrets_test . Env

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

type Env interface {
	ListFederationSecrets(ctx context.Context) ([]db.ListFederationSecretsRow, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env) ([]db.ListFederationSecretsRow, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	rows, err := env.ListFederationSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list federation secrets: %w", err)
	}

	return rows, nil
}
