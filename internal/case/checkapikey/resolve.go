package checkapikey

import (
	"context"
	"errors"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
)

type Env interface {
	ApiKeyByValue(ctx context.Context, value string) (db.ApiKey, error)
	InsertApiKeyLog(ctx context.Context, arg db.InsertApiKeyLogParams) error
	UpsertApiKeyLogAction(ctx context.Context, name string) error
	UpsertApiKeyLogIP(ctx context.Context, ip string) error
}

var ErrMissingKey = errors.New("missing API key in request header")
var ErrInvalidKey = errors.New("invalid API key")

func Resolve(ctx context.Context, env Env, action string) (*db.ApiKey, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get request from context: %w", err)
	}

	apiKeyValue := req.Req.Request.Header.Peek("X-API-Key")
	if len(apiKeyValue) == 0 {
		return nil, ErrMissingKey
	}

	apiKey, err := env.ApiKeyByValue(ctx, string(apiKeyValue))
	if err != nil {
		if db.IsNoFound(err) {
			return nil, ErrInvalidKey
		}

		return nil, fmt.Errorf("failed to resolve API key: %w", err)
	}

	err = env.UpsertApiKeyLogAction(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert API key log action: %w", err)
	}

	ip := req.Req.RemoteIP().String()

	err = env.UpsertApiKeyLogIP(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert API key log IP: %w", err)
	}

	params := db.InsertApiKeyLogParams{
		ApiKeyID: apiKey.ID,
		Action:   action,
		Ip:       ip,
	}

	err = env.InsertApiKeyLog(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert API key log: %w", err)
	}

	return &apiKey, nil
}
