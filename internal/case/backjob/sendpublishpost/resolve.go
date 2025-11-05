package sendpublishpost

import (
	"context"
	"trip2g/internal/model"
)

type Env interface {
	SendTelegramPublishPostWithTx(ctx context.Context, params model.SendTelegramPublishPostParams) error
}

func Resolve(ctx context.Context, env Env, params model.SendTelegramPublishPostParams) error {
	return env.SendTelegramPublishPostWithTx(ctx, params)
}
