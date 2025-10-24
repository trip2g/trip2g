package convertnoteviewtotgpost

import (
	"context"
	"trip2g/internal/markdownv2"
	"trip2g/internal/model"
)

type Env interface {
}

func Resolve(ctx context.Context, env Env, nv *model.NoteView) (*model.TelegramPost, error) {
	tr := markdownv2.CommonConverter{}
	res := tr.Process(nv)

	return &model.TelegramPost{
		Content:  res.Content,
		Warnings: res.Warnings,
	}, nil
}
