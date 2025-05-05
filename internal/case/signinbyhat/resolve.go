package signinbyhat

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

type Env interface {
	ParseHotAuthToken(_ context.Context, token string) (*model.HotAuthToken, error)
	SetupUserToken(ctx context.Context, userID int64) (string, error)
	UserByEmail(ctx context.Context, email string) (db.User, error)
}

func Resolve(ctx context.Context, env Env, rawToken string) error {
	token, err := env.ParseHotAuthToken(ctx, rawToken)
	if err != nil {
		return fmt.Errorf("failed to parse hot auth token: %w", err)
	}

	user, err := env.UserByEmail(ctx, token.Email)
	if err != nil {
		return fmt.Errorf("failed to find user by email: %w", err)
	}

	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to setup user token: %w", err)
	}

	return nil
}
