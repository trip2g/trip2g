package setsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg setsecret_test . Env

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	IsDevMode() bool
	WebhookByID(ctx context.Context, id int64) (db.ChangeWebhook, error)
	CronWebhookByID(ctx context.Context, id int64) (db.CronWebhook, error)
	UpsertSecret(ctx context.Context, arg db.UpsertSecretParams) (db.Secret, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
}

// webhookTarget parses keys in the model.SecretPrefix format
// ("change_webhooks:<id>:<name>" / "cron_webhooks:<id>:<name>").
func webhookTarget(key string) (string, int64, bool) {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) != 3 {
		return "", 0, false
	}
	if parts[0] != "change_webhooks" && parts[0] != "cron_webhooks" {
		return "", 0, false
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		return "", 0, false
	}
	return parts[0], id, true
}

// validateWebhookURL rejects attaching a secret to a webhook whose URL is not
// HTTPS: delivery would send the decrypted secret over cleartext. Same rule as
// create/update webhook with pass_api_key.
func validateWebhookURL(ctx context.Context, env Env, key string) (*model.ErrorPayload, error) {
	entity, id, ok := webhookTarget(key)
	if !ok {
		return nil, nil
	}

	var whURL string
	switch entity {
	case "change_webhooks":
		wh, err := env.WebhookByID(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return model.NewFieldError("key", fmt.Sprintf("webhook %d not found", id)), nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load webhook %d: %w", id, err)
		}
		whURL = wh.Url
	case "cron_webhooks":
		wh, err := env.CronWebhookByID(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return model.NewFieldError("key", fmt.Sprintf("cron webhook %d not found", id)), nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load cron webhook %d: %w", id, err)
		}
		whURL = wh.Url
	}

	if msg := webhookutil.RequireHTTPSUnlessDevMode(whURL, env.IsDevMode()); msg != "" {
		return model.NewFieldError("key", msg), nil
	}
	return nil, nil
}

func Resolve(ctx context.Context, env Env, key, value string) (*model.ErrorPayload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	if errPayload, verr := validateWebhookURL(ctx, env, key); errPayload != nil || verr != nil {
		return errPayload, verr
	}

	encrypted, err := env.EncryptData([]byte(value))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	_, err = env.UpsertSecret(ctx, db.UpsertSecretParams{
		Key:        key,
		ValueCrypt: encrypted,
		CreatedBy:  int64(token.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert secret: %w", err)
	}

	return nil, nil
}
