package checkapikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	ApiKeyByValue(ctx context.Context, value string) (db.ApiKey, error)
	InsertAPIKeyLog(ctx context.Context, arg db.InsertAPIKeyLogParams) error
	UpsertAPIKeyLogAction(ctx context.Context, name string) error
	UpsertAPIKeyLogIP(ctx context.Context, ip string) error
	ShortAPITokenSecret() string
}

var ErrMissingKey = errors.New("missing X-API-Key in request header")
var ErrInvalidKey = errors.New("invalid API key")
var ErrInvalidToken = errors.New("invalid or expired Bearer token")

func Resolve(ctx context.Context, env Env, action string) (*db.ApiKey, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get request from context: %w", err)
	}

	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	if token.IsAdmin() {
		return &db.ApiKey{
			ID:          0,
			Value:       "admin",
			Description: "Admin user bypass",
		}, nil
	}

	apiKeyValue := req.Req.Request.Header.Peek("X-API-Key")
	if len(apiKeyValue) > 0 {
		return resolveAPIKey(ctx, env, string(apiKeyValue), action, req)
	}

	// Try Authorization: Bearer {shortapitoken}.
	authHeader := req.Req.Request.Header.Peek("Authorization")
	if len(authHeader) > 0 {
		const bearerPrefix = "Bearer "
		authStr := string(authHeader)
		if strings.HasPrefix(authStr, bearerPrefix) {
			tokenStr := authStr[len(bearerPrefix):]
			return resolveShortAPIToken(ctx, env, tokenStr, req)
		}
	}

	return nil, ErrMissingKey
}

func resolveAPIKey(ctx context.Context, env Env, plainKey string, action string, req *appreq.Request) (*db.ApiKey, error) {
	// First, try hashed version (new API keys).
	hash := sha256.Sum256([]byte(plainKey))
	hashedValue := hex.EncodeToString(hash[:])

	apiKey, err := env.ApiKeyByValue(ctx, hashedValue)
	if err != nil && !db.IsNoFound(err) {
		return nil, fmt.Errorf("failed to resolve API key: %w", err)
	}

	// Backward compatibility: try plain text (old API keys) if hashed not found.
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

	req.SkipWebhooks = apiKey.SkipWebhooks

	return &apiKey, nil
}

//nolint:unparam // ctx kept for interface consistency.
func resolveShortAPIToken(ctx context.Context, env Env, tokenStr string, req *appreq.Request) (*db.ApiKey, error) {
	data, err := shortapitoken.Parse(tokenStr, env.ShortAPITokenSecret())
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Store auth data in Request for downstream code.
	req.WebhookDepth = data.Depth
	req.WebhookReadPatterns = data.ReadPatterns
	req.WebhookWritePatterns = data.WritePatterns

	// Return a virtual ApiKey for shortapitoken auth.
	virtualKey := db.ApiKey{
		ID:           0, // Virtual key, no DB record.
		Value:        "shortapitoken",
		Description:  "Short API token (webhook)",
		SkipWebhooks: false, // shortapitoken should never skip webhooks.
	}

	return &virtualKey, nil
}
