package disableapikey

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	DisableApiKey(ctx context.Context, params db.DisableApiKeyParams) (db.ApiKey, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.DisableAPIKeyInput
type Payload = model.DisableAPIKeyOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	params := db.DisableApiKeyParams{
		ID:         input.ID,
		DisabledBy: ptr.To(int64(token.ID)),
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
