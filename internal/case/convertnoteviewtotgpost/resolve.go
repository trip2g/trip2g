package convertnoteviewtotgpost

import (
	"context"
	"trip2g/internal/model"
)

type Env interface {
}

func Resolve(ctx context.Context, env Env, nv *model.NoteView) (*model.TelegramPost, error) {
	return &model.TelegramPost{}, nil
}
