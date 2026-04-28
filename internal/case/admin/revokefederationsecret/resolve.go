package revokefederationsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg revokefederationsecret_test . Env

import (
	"context"
	"fmt"

	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	RevokeFederationSecret(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, id int64) (model.RevokeFederationSecretOrErrorPayload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	err = env.RevokeFederationSecret(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke federation secret: %w", err)
	}

	return &model.RevokeFederationSecretPayload{RevokedID: id}, nil
}
