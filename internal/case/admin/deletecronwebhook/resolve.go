package deletecronwebhook

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	DisableCronWebhook(ctx context.Context, params db.DisableCronWebhookParams) error
}

type Input = model.DeleteCronWebhookInput
type Payload = model.DeleteCronWebhookOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	params := db.DisableCronWebhookParams{
		ID:         input.ID,
		DisabledBy: ptr.To(int64(token.ID)),
	}

	err = env.DisableCronWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to disable cron webhook: %w", err)
	}

	return &model.DeleteCronWebhookPayload{
		DeletedID: input.ID,
	}, nil
}
