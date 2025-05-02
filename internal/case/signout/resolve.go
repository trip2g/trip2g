package signout

import (
	"context"
	"fmt"
	gmodel "trip2g/internal/graph/model"

	"trip2g/internal/model"
)

type Env interface {
	ResetUserToken(ctx context.Context) error
}

func Resolve(ctx context.Context, env Env) (gmodel.SignOutOrErrorPayload, error) {
	err := env.ResetUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to reset user token: %w", err)
	}

	response := gmodel.SignOutPayload{
		Viewer: &model.Viewer{},
	}

	return &response, nil
}
