package main

import (
	"context"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/case/sendtelegramaccountpublishpost"
	"trip2g/internal/case/sendtelegrampublishpost"
	"trip2g/internal/case/updatetelegramaccountpublishpost"
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

// Account publishing methods.
func (a *app) UpdateTelegramAccountPublishPost(ctx context.Context, notePathID int64) error {
	return updatetelegramaccountpublishpost.Resolve(ctx, a, notePathID)
}

func (a *app) SendTelegramAccountPublishPost(ctx context.Context, params model.SendTelegramPublishPostParams) error {
	return sendtelegramaccountpublishpost.Resolve(ctx, a, params)
}

func (a *app) SendTelegramAccountPublishPostWithTx(ctx context.Context, params model.SendTelegramPublishPostParams) error {
	return a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		err := sendtelegramaccountpublishpost.Resolve(txCtx, env, params)
		return err == nil, err
	})
}

func (a *app) UpdateTelegramAccountPublishPostWithTx(ctx context.Context, notePathID int64) error {
	return a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		err := updatetelegramaccountpublishpost.Resolve(txCtx, env, notePathID)
		return err == nil, err
	})
}
