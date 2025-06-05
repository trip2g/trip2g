package deleteredirect

import (
	"context"
	"fmt"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DeleteRedirect(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, input model.DeleteRedirectInput) (model.DeleteRedirectOrErrorPayload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	err = env.DeleteRedirect(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete redirect: %w", err)
	}

	response := model.DeleteRedirectPayload{
		ID: input.ID,
	}

	return &response, nil
}