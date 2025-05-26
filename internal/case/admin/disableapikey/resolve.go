package disableapikey

import (
	"context"
	"database/sql"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	DisableApiKey(ctx context.Context, params db.DisableApiKeyParams) (db.ApiKey, error)
}

func Resolve(ctx context.Context, env Env, input model.DisableAPIKeyInput) (model.DisableAPIKeyOrErrorPayload, error) {
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

	params := db.DisableApiKeyParams{
		ID:         int64(input.ID),
		DisabledBy: sql.NullInt64{Valid: true, Int64: int64(token.ID)},
	}

	apiKey, err := env.DisableApiKey(ctx, params)
	if err != nil {
		return nil, err
	}

	response := model.DisableAPIKeyPayload{
		APIKey: &apiKey,
	}

	return &response, nil
}
