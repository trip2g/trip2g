package main

import (
	"context"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/case/getpublicnote"
	"trip2g/internal/case/gettelegramchatname"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/case/listnotepaths"
	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/case/resolvewikilinks"
	"trip2g/internal/case/sendtelegramaccountpublishpost"
	"trip2g/internal/case/sendtelegrampublishpost"
	"trip2g/internal/case/sitesearch"
	"trip2g/internal/case/updatetelegramaccountpublishpost"
	"trip2g/internal/case/updatetelegrampublishpost"
	"trip2g/internal/case/vieweroffers"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/model"
)

var (
	_ getpublicnote.Env    = (*app)(nil)
	_ vieweroffers.Env     = (*app)(nil)
	_ listnotepaths.Env    = (*app)(nil)
	_ resolvewikilinks.Env = (*app)(nil)
)

// RenderNotePage is the opaque cross-case port used by getpublicnote to render
// a note page without asserting the concrete rendernotepage.Env.
func (a *app) RenderNotePage(ctx context.Context, request rendernotepage.Request) (*rendernotepage.Response, error) {
	return rendernotepage.Resolve(ctx, a, request)
}

// SearchNotes is the opaque cross-case port used by listnotepaths to run a site
// search without asserting the concrete sitesearch.Env.
func (a *app) SearchNotes(ctx context.Context, input graphmodel.SearchInput) (*graphmodel.SearchConnection, error) {
	return sitesearch.Resolve(ctx, a, input)
}

func (a *app) HandleNoteWebhooks(ctx context.Context, changes []handlenotewebhooks.NoteChange, depth int) error {
	return handlenotewebhooks.Resolve(ctx, a, changes, depth)
}

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

func (a *app) GetTelegramChatName(ctx context.Context, telegramChatID int64) (string, error) {
	return gettelegramchatname.Resolve(ctx, a, telegramChatID)
}

func (a *app) RefreshStaleTelegramChatUsernames(ctx context.Context, limit int) (int, error) {
	return gettelegramchatname.RefreshStale(ctx, a, limit)
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
