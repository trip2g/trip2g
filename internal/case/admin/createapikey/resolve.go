package createapikey

import (
	"context"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	GenerateApiKey() string
	InsertApiKey(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error)
}

func Resolve(ctx context.Context, env Env, input model.CreateAPIKeyInput) (model.CreateAPIKeyOrErrorPayload, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	if !token.IsAdmin() {
		return &model.ErrorPayload{Message: "Unauthorized"}, nil
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
