package disableapikey

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DisableApiKey(ctx context.Context, params db.DisableApiKeyParams) (db.ApiKey, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, input model.DisableAPIKeyInput) (model.DisableAPIKeyOrErrorPayload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
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
