package disableapikey

import (
	"context"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"database/sql"
)

type Env interface {
	DisableApiKey(ctx context.Context, params db.DisableApiKeyParams) error
}

func Resolve(ctx context.Context, env Env, input model.DeleteAPIKeyInput) (model.DeleteAPIKeyOrErrorPayload, error) {
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

	err = env.DisableApiKey(ctx, params)
	if err != nil {
		return nil, err
	}

	response := model.DeleteAPIKeyPayload{
		ID: input.ID,
	}

	return &response, nil
}