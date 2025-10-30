package checkapikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
)

type Env interface {
	ApiKeyByValue(ctx context.Context, value string) (db.ApiKey, error)
	InsertAPIKeyLog(ctx context.Context, arg db.InsertAPIKeyLogParams) error
	UpsertAPIKeyLogAction(ctx context.Context, name string) error
	UpsertAPIKeyLogIP(ctx context.Context, ip string) error
}

var ErrMissingKey = errors.New("missing X-API-Key in request header")
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

	plainKey := string(apiKeyValue)

	// First, try hashed version (new API keys)
	hash := sha256.Sum256([]byte(plainKey))
	hashedValue := hex.EncodeToString(hash[:])

	apiKey, err := env.ApiKeyByValue(ctx, hashedValue)
	if err != nil && !db.IsNoFound(err) {
		return nil, fmt.Errorf("failed to resolve API key: %w", err)
	}

	// Backward compatibility: try plain text (old API keys) if hashed not found
	if db.IsNoFound(err) {
		apiKey, err = env.ApiKeyByValue(ctx, plainKey)
		if err != nil && !db.IsNoFound(err) {
			return nil, fmt.Errorf("failed to resolve API key: %w", err)
		}

		if db.IsNoFound(err) {
			return nil, ErrInvalidKey
		}
	}

	err = env.UpsertAPIKeyLogAction(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert API key log action: %w", err)
	}

	ip := req.Req.RemoteIP().String()

	err = env.UpsertAPIKeyLogIP(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert API key log IP: %w", err)
	}

	params := db.InsertAPIKeyLogParams{
		ApiKeyID: apiKey.ID,
		Action:   action,
		Ip:       ip,
	}

	err = env.InsertAPIKeyLog(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert API key log: %w", err)
	}

	return &apiKey, nil
}
