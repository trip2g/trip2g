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

func (a *app) SendTelegramPublishPost(ctx context.Context, params model.SendTelegramPublishPostParams) error {
	return sendtelegrampublishpost.Resolve(ctx, a, params)
}

func (a *app) SendTelegramPublishPostWithTx(ctx context.Context, params model.SendTelegramPublishPostParams) error {
	return a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		err := sendtelegrampublishpost.Resolve(txCtx, env, params)
		return err == nil, err
	})
}

func (a *app) UpdateTelegramPublishPostWithTx(ctx context.Context, notePathID int64) error {
	return a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		err := updatetelegrampublishpost.Resolve(txCtx, env, notePathID)
		return err == nil, err
	})
}

func (a *app) ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
	return convertnoteviewtotgpost.Resolve(ctx, a, source)
}
