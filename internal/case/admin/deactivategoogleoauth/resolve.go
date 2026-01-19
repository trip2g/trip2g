package deactivategoogleoauth

import (
	"context"
	"fmt"

	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DeactivateAllGoogleOAuthCredentials(ctx context.Context) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Payload = model.DeactivateGoogleOAuthOrErrorPayload

func Resolve(ctx context.Context, env Env) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	err = env.DeactivateAllGoogleOAuthCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate google oauth: %w", err)
	}

	payload := model.DeactivateGoogleOAuthPayload{
		Success: true,
	}

	return &payload, nil
}
