package vieweroffers

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type Env interface {
	GetLatestConfigBool(ctx context.Context, valueID string) (db.GetLatestConfigBoolRow, error)
	LatestNoteViews() *appmodel.NoteViews
	LiveNoteViews() *appmodel.NoteViews
	ListActiveOffersBySubgraphNames(ctx context.Context, subgraphs []string) ([]db.Offer, error)
	ListEnabledTgBots(ctx context.Context) ([]db.TgBot, error)
}

func Resolve(ctx context.Context, env Env, filter model.ViewerOffersFilter) (model.ViewerOffers, error) {
	if filter.PageID == nil {
		return nil, nil
	}

	showDraftVersions := false
	if entry, err := env.GetLatestConfigBool(ctx, "show_draft_versions"); err == nil {
		showDraftVersions = entry.Value
	}

	var note *appmodel.NoteView

	if showDraftVersions {
		note = env.LatestNoteViews().GetByPathID(*filter.PageID)
	} else {
		note = env.LiveNoteViews().GetByPathID(*filter.PageID)
	}

	if note == nil {
		return nil, nil
	}

	subgraphNames := note.SubgraphNames
	if len(subgraphNames) == 0 {
		subgraphs := env.LiveNoteViews().Subgraphs

		for name := range subgraphs {
			subgraphNames = append(subgraphNames, name)
		}
	}

	offers, err := env.ListActiveOffersBySubgraphNames(ctx, subgraphNames)
	if err != nil {
		return nil, err
	}

	if len(offers) > 0 {
		return &model.ActiveOffers{Nodes: offers}, nil
	}

	// Nothing to sell: only sites that opted into wait lists collect contacts here.
	showWaitlists := false

	entry, cfgErr := env.GetLatestConfigBool(ctx, "show_waitlists")
	if cfgErr == nil {
		showWaitlists = entry.Value
	}

	if !showWaitlists {
		return nil, nil
	}

	wl := model.SubgraphWaitList{
		EmailAllowed: true,
	}

	bots, err := env.ListEnabledTgBots(ctx)
	if err != nil {
		return nil, err
	}

	if len(bots) > 0 {
		botURL := fmt.Sprintf("https://t.me/%s?start=wl_%d", bots[0].Name, note.PathID)

		wl.TgBotURL = &botURL
	}

	return &wl, nil
}
