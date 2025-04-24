package signout

import (
	"context"
	"fmt"
	"trip2g/internal/graph/model"
)

type Env interface {
	ResetUserToken(ctx context.Context) error
}

func Resolve(ctx context.Context, env Env) (model.SignOutOrErrorPayload, error) {
	err := env.ResetUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to reset user token: %w", err)
	}

	response := model.SignOutPayload{
		Viewer: &model.Viewer{},
	}

	return &response, nil
}
