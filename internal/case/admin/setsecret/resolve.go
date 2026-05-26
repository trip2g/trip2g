package setsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg setsecret_test . Env

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

type Env interface {
	UpsertSecret(ctx context.Context, arg db.UpsertSecretParams) (db.Secret, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
}

func Resolve(ctx context.Context, env Env, key, value string) error {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current admin user token: %w", err)
	}

	encrypted, err := env.EncryptData([]byte(value))
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	_, err = env.UpsertSecret(ctx, db.UpsertSecretParams{
		Key:        key,
		ValueCrypt: encrypted,
		CreatedBy:  int64(token.ID),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert secret: %w", err)
	}

	return nil
}
