package deactivategithuboauth

import (
	"context"
	"fmt"

	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DeactivateAllGitHubOAuthCredentials(ctx context.Context) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Payload = model.DeactivateGitHubOAuthOrErrorPayload

func Resolve(ctx context.Context, env Env) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	err = env.DeactivateAllGitHubOAuthCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate github oauth: %w", err)
	}

	payload := model.DeactivateGitHubOAuthPayload{
		Success: true,
	}

	return &payload, nil
}
