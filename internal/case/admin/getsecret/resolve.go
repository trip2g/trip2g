package getsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg getsecret_test . Env

import (
	"context"
	"fmt"

	"trip2g/internal/db"
)

type Env interface {
	GetSecret(ctx context.Context, key string) (db.Secret, error)
	DecryptData(ciphertext []byte) ([]byte, error)
}

func Resolve(ctx context.Context, env Env, key string) (string, error) {
	row, err := env.GetSecret(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	plain, err := env.DecryptData(row.ValueCrypt)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return string(plain), nil
}
