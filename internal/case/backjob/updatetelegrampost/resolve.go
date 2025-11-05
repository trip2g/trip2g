package updatetelegrampost

import (
	"context"
)

type Env interface {
	UpdateTelegramPublishPostWithTx(ctx context.Context, notePathID int64) error
}

func Resolve(ctx context.Context, env Env, notePathID int64) error {
	return env.UpdateTelegramPublishPostWithTx(ctx, notePathID)
}
