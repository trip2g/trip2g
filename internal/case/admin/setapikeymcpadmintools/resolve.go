package setapikeymcpadmintools

import (
	"context"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg setapikeymcpadmintools_test . Env

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	SetApiKeyMcpAdminTools(ctx context.Context, arg db.SetApiKeyMcpAdminToolsParams) error
	ApiKeyByID(ctx context.Context, id int64) (db.ApiKey, error)
}

type Input = model.SetAPIKeyMcpAdminToolsInput
type Payload = model.SetAPIKeyMcpAdminToolsOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	if _, err := env.CurrentAdminUserToken(ctx); err != nil {
		return nil, err
	}
	if err := env.SetApiKeyMcpAdminTools(ctx, db.SetApiKeyMcpAdminToolsParams{
		ID:      input.ID,
		Enabled: &input.Enabled,
	}); err != nil {
		return nil, err
	}
	key, err := env.ApiKeyByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return &model.SetAPIKeyMcpAdminToolsPayload{APIKey: &key}, nil
}
