package deletewebhook

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
	DisableWebhook(ctx context.Context, params db.DisableWebhookParams) error
}

type Input = model.ChangeWebhookDeleteInput
type Payload = model.ChangeWebhookDeleteOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	params := db.DisableWebhookParams{
		ID:         input.ID,
		DisabledBy: ptr.To(int64(token.ID)),
	}

	err = env.DisableWebhook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to disable webhook: %w", err)
	}

	return &model.ChangeWebhookDeletePayload{
		DeletedID: input.ID,
	}, nil
}
