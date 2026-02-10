package regeneratewebhooksecret

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	RegenerateWebhookSecret(ctx context.Context, params db.RegenerateWebhookSecretParams) (db.ChangeWebhook, error)
}

type Input = model.RegenerateWebhookSecretInput
type Payload = model.RegenerateWebhookSecretOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	secret, err := webhookutil.GenerateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	params := db.RegenerateWebhookSecretParams{
		ID:     input.ID,
		Secret: secret,
	}

	webhook, err := env.RegenerateWebhookSecret(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate webhook secret: %w", err)
	}

	return &model.RegenerateWebhookSecretPayload{
		Webhook: &webhook,
		Secret:  secret,
	}, nil
}
