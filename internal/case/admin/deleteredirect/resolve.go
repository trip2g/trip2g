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

type Input = model.DeleteRedirectInput
type Payload = model.DeleteRedirectOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	err = env.DeleteRedirect(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete redirect: %w", err)
	}

	response := model.DeleteRedirectPayload(input)

	return &response, nil
}
