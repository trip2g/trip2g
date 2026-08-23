package vieweroffers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/vieweroffers"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func ptr[T any](v T) *T { return &v }

func configBool(values map[string]bool) func(context.Context, string) (db.GetLatestConfigBoolRow, error) {
	return func(_ context.Context, valueID string) (db.GetLatestConfigBoolRow, error) {
		return db.GetLatestConfigBoolRow{Value: values[valueID]}, nil
	}
}

func viewsWith(note *appmodel.NoteView) *appmodel.NoteViews {
	nv := &appmodel.NoteViews{Map: map[string]*appmodel.NoteView{}}
	if note != nil {
		nv.List = append(nv.List, note)
		nv.Map[note.Permalink] = note
	}
	return nv
}

func TestResolve_NilPageID(t *testing.T) {
	env := &vieweroffers.EnvMock{}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: nil})
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestResolve_NoteNotFound(t *testing.T) {
	env := &vieweroffers.EnvMock{
		GetLatestConfigBoolFunc: func(ctx context.Context, valueID string) (db.GetLatestConfigBoolRow, error) {
			return db.GetLatestConfigBoolRow{Value: false}, nil
		},
		LiveNoteViewsFunc: func() *appmodel.NoteViews { return viewsWith(nil) },
	}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: ptr(int64(5))})
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestResolve_DraftSelectsLatest(t *testing.T) {
	note := &appmodel.NoteView{PathID: 5, SubgraphNames: []string{"g1"}}
	usedLatest := false
	env := &vieweroffers.EnvMock{
		GetLatestConfigBoolFunc: func(ctx context.Context, valueID string) (db.GetLatestConfigBoolRow, error) {
			return db.GetLatestConfigBoolRow{Value: true}, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews { usedLatest = true; return viewsWith(note) },
		LiveNoteViewsFunc:   func() *appmodel.NoteViews { return viewsWith(nil) },
		ListActiveOffersBySubgraphNamesFunc: func(ctx context.Context, subgraphs []string) ([]db.Offer, error) {
			require.Equal(t, []string{"g1"}, subgraphs)
			return []db.Offer{{ID: 1}}, nil
		},
	}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: ptr(int64(5))})
	require.NoError(t, err)
	require.True(t, usedLatest)
	ao, ok := out.(*model.ActiveOffers)
	require.True(t, ok)
	require.Len(t, ao.Nodes, 1)
}

func TestResolve_LiveSelectionAndSubgraphFallback(t *testing.T) {
	// Note has no subgraph names -> fall back to all live subgraphs.
	note := &appmodel.NoteView{PathID: 5}
	live := viewsWith(note)
	live.Subgraphs = map[string]*appmodel.NoteSubgraph{"all": {}}
	var gotSubgraphs []string
	env := &vieweroffers.EnvMock{
		GetLatestConfigBoolFunc: func(ctx context.Context, valueID string) (db.GetLatestConfigBoolRow, error) {
			return db.GetLatestConfigBoolRow{Value: false}, nil
		},
		LiveNoteViewsFunc: func() *appmodel.NoteViews { return live },
		ListActiveOffersBySubgraphNamesFunc: func(ctx context.Context, subgraphs []string) ([]db.Offer, error) {
			gotSubgraphs = subgraphs
			return []db.Offer{{ID: 2}}, nil
		},
	}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: ptr(int64(5))})
	require.NoError(t, err)
	require.Equal(t, []string{"all"}, gotSubgraphs)
	_, ok := out.(*model.ActiveOffers)
	require.True(t, ok)
}

func TestResolve_WaitlistWithBot(t *testing.T) {
	note := &appmodel.NoteView{PathID: 42, SubgraphNames: []string{"g1"}}
	env := &vieweroffers.EnvMock{
		GetLatestConfigBoolFunc: configBool(map[string]bool{"show_waitlists": true}),
		LiveNoteViewsFunc:       func() *appmodel.NoteViews { return viewsWith(note) },
		ListActiveOffersBySubgraphNamesFunc: func(ctx context.Context, subgraphs []string) ([]db.Offer, error) {
			return nil, nil
		},
		ListEnabledTgBotsFunc: func(ctx context.Context) ([]db.TgBot, error) {
			return []db.TgBot{{Name: "mybot"}}, nil
		},
	}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: ptr(int64(42))})
	require.NoError(t, err)
	wl, ok := out.(*model.SubgraphWaitList)
	require.True(t, ok)
	require.True(t, wl.EmailAllowed)
	require.NotNil(t, wl.TgBotURL)
	require.Equal(t, "https://t.me/mybot?start=wl_42", *wl.TgBotURL)
}

func TestResolve_WaitlistWithoutBots(t *testing.T) {
	note := &appmodel.NoteView{PathID: 42, SubgraphNames: []string{"g1"}}
	env := &vieweroffers.EnvMock{
		GetLatestConfigBoolFunc: configBool(map[string]bool{"show_waitlists": true}),
		LiveNoteViewsFunc:       func() *appmodel.NoteViews { return viewsWith(note) },
		ListActiveOffersBySubgraphNamesFunc: func(ctx context.Context, subgraphs []string) ([]db.Offer, error) {
			return nil, nil
		},
		ListEnabledTgBotsFunc: func(ctx context.Context) ([]db.TgBot, error) {
			return nil, nil
		},
	}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: ptr(int64(42))})
	require.NoError(t, err)
	wl, ok := out.(*model.SubgraphWaitList)
	require.True(t, ok)
	require.True(t, wl.EmailAllowed)
	require.Nil(t, wl.TgBotURL)
}

func TestResolve_OffersError(t *testing.T) {
	note := &appmodel.NoteView{PathID: 5, SubgraphNames: []string{"g1"}}
	env := &vieweroffers.EnvMock{
		GetLatestConfigBoolFunc: func(ctx context.Context, valueID string) (db.GetLatestConfigBoolRow, error) {
			return db.GetLatestConfigBoolRow{Value: false}, nil
		},
		LiveNoteViewsFunc: func() *appmodel.NoteViews { return viewsWith(note) },
		ListActiveOffersBySubgraphNamesFunc: func(ctx context.Context, subgraphs []string) ([]db.Offer, error) {
			return nil, errors.New("db down")
		},
	}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: ptr(int64(5))})
	require.Error(t, err)
	require.Nil(t, out)
}

func TestResolve_WaitlistDisabled(t *testing.T) {
	note := &appmodel.NoteView{PathID: 42, SubgraphNames: []string{"g1"}}
	env := &vieweroffers.EnvMock{
		GetLatestConfigBoolFunc: configBool(nil),
		LiveNoteViewsFunc:       func() *appmodel.NoteViews { return viewsWith(note) },
		ListActiveOffersBySubgraphNamesFunc: func(ctx context.Context, subgraphs []string) ([]db.Offer, error) {
			return nil, nil
		},
	}
	out, err := vieweroffers.Resolve(context.Background(), env, model.ViewerOffersFilter{PageID: ptr(int64(42))})
	require.NoError(t, err)
	require.Nil(t, out)
}
