package enableapikey

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	EnableApiKey(ctx context.Context, id int64) (db.ApiKey, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.EnableAPIKeyInput
type Payload = model.EnableAPIKeyOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	apiKey, err := env.EnableApiKey(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	return &model.EnableAPIKeyPayload{APIKey: &apiKey}, nil
}
