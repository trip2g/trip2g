package signinbytgauthtoken

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	SetupUserToken(ctx context.Context, userID int64) (string, error)
	ParseTgAuthToken(_ context.Context, token string) (*model.TgAuthToken, error)
}

func Resolve(ctx context.Context, env Env, rawToken string) error {
	_, err := env.ParseTgAuthToken(ctx, rawToken)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	// TODO: make user if needed

	_, err = env.SetupUserToken(ctx, 0)
	if err != nil {
		return fmt.Errorf("failed to setup user token: %w", err)
	}

	return nil
}
