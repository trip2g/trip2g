package regeneratecronwebhooksecret

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
	RegenerateCronWebhookSecret(ctx context.Context, params db.RegenerateCronWebhookSecretParams) (db.CronWebhook, error)
}

type Input = model.RegenerateCronWebhookSecretInput
type Payload = model.RegenerateCronWebhookSecretOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	secret, err := webhookutil.GenerateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	params := db.RegenerateCronWebhookSecretParams{
		ID:     input.ID,
		Secret: secret,
	}

	webhook, err := env.RegenerateCronWebhookSecret(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate cron webhook secret: %w", err)
	}

	return &model.RegenerateCronWebhookSecretPayload{
		CronWebhook: &webhook,
		Secret:      secret,
	}, nil
}
