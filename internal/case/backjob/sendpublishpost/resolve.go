package sendpublishpost

import (
	"context"
)

type Params struct {
	NotePathID int64 `json:"note_path_id"`
	Instant    bool  `json:"instant"`
}

type Env interface {
	SendTelegramPublishPostWithTx(ctx context.Context, notePathID int64, instant bool) error
}

func Resolve(ctx context.Context, env Env, params Params) error {
	return env.SendTelegramPublishPostWithTx(ctx, params.NotePathID, params.Instant)
}
