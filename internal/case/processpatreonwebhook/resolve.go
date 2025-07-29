package processpatreonwebhook

import (
	"context"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // it's okay to use MD5 here as per Patreon documentation
	"encoding/hex"
	"errors"
	"fmt"

	"trip2g/internal/case/refreshpatreondata"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	refreshpatreondata.Env
}

type Request struct {
	CredentialID int64  `json:"credential_id"`
	Signature    string `json:"signature"`
	Body         []byte `json:"body"`
}

type Response struct {
	Success bool `json:"success"`
}

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	// Get credentials to access webhook secret and verify existence
	credentials, err := env.PatreonCredentials(ctx, request.CredentialID)
	if err != nil {
		if db.IsNoFound(err) {
			env.Logger().Warn("credentials not found", "credential_id", request.CredentialID)
			return nil, errors.New("credentials not found")
		}
		return nil, fmt.Errorf("failed to get patreon credentials: %w", err)
	}

	// Check if webhook secret exists
	if !credentials.WebhookSecret.Valid || credentials.WebhookSecret.String == "" {
		env.Logger().Error("webhook secret not configured for credential", "credential_id", request.CredentialID)
		return nil, errors.New("webhook secret not configured")
	}

	// Verify webhook signature
	if !verifyWebhookSignature(request.Body, credentials.WebhookSecret.String, request.Signature) {
		env.Logger().Error("invalid webhook signature", "credential_id", request.CredentialID)
		return nil, errors.New("invalid webhook signature")
	}

	env.Logger().Info("processing patreon webhook",
		"credential_id", request.CredentialID,
	)

	// Call refreshpatreondata to sync the data
	err = refreshpatreondata.Resolve(ctx, env, &credentials.ID)
	if err != nil {
		env.Logger().Error("failed to refresh patreon data", "error", err, "credential_id", request.CredentialID)
		return nil, fmt.Errorf("failed to refresh patreon data: %w", err)
	}

	return &Response{Success: true}, nil
}

// verifyWebhookSignature verifies the Patreon webhook signature
// According to Patreon docs: HEX digest of the message body HMAC signed (with MD5) using webhook secret.
func verifyWebhookSignature(body []byte, secret, signature string) bool {
	h := hmac.New(md5.New, []byte(secret))
	h.Write(body)
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}
