package signout

import (
	"context"
	"fmt"
	gmodel "trip2g/internal/graph/model"

	"trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg signout_test . Env

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
