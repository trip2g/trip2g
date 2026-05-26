package deletesecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg deletesecret_test . Env

import (
	"context"
	"fmt"

	"trip2g/internal/usertoken"
)

type Env interface {
	DeleteSecret(ctx context.Context, key string) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, key string) error {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current admin user token: %w", err)
	}

	err = env.DeleteSecret(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}
