package updatetelegramaccountpost

import (
	"context"
)

type Env interface {
	UpdateTelegramAccountPublishPostWithTx(ctx context.Context, notePathID int64) error
}

func Resolve(ctx context.Context, env Env, notePathID int64) error {
	return env.UpdateTelegramAccountPublishPostWithTx(ctx, notePathID)
}
