package createapikey

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	GenerateApiKey() string
	InsertApiKey(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error)
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, input model.CreateAPIKeyInput) (model.CreateAPIKeyOrErrorPayload, error) {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	apiKey := env.GenerateApiKey()

	params := db.InsertApiKeyParams{
		Value:       apiKey,
		CreatedBy:   int64(token.ID),
		Description: input.Description,
	}

	createdKey, err := env.InsertApiKey(ctx, params)
	if err != nil {
		return nil, err
	}

	response := model.CreateAPIKeyPayload{
		Value:  apiKey,
		APIKey: &createdKey,
	}

	return &response, nil
}
