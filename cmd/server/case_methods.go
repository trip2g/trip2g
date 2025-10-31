package main

import (
	"context"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/case/sendtelegrampublishpost"
	"trip2g/internal/case/updatetelegrampublishpost"
	"trip2g/internal/model"
)

func (a *app) UpdateTelegramPublishPost(ctx context.Context, notePathID int64) error {
	return updatetelegrampublishpost.Resolve(ctx, a, notePathID)
}

func (a *app) SendTelegramPublishPost(ctx context.Context, notePathID int64, instant bool) error {
	return sendtelegrampublishpost.Resolve(ctx, a, notePathID, instant)
}

func (a *app) SendTelegramPublishPostWithTx(ctx context.Context, notePathID int64, instant bool) error {
	return a.WithTransaction(ctx, func(env *app) (bool, error) {
		err := sendtelegrampublishpost.Resolve(ctx, env, notePathID, instant)
		return err != nil, err
	})
}

func (a *app) ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
	return convertnoteviewtotgpost.Resolve(ctx, a, source)
}
