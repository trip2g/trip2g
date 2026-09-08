package rendernotepage_test

import (
	"context"
	"testing"

	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func teaserAccessNotes() (*model.NoteViews, map[string]*model.NoteView) {
	mk := func(path, permalink string, sgs map[string]*model.NoteSubgraph) *model.NoteView {
		names := make([]string, 0, len(sgs))
		for name := range sgs {
			names = append(names, name)
		}
		return &model.NoteView{
			Path: path, Permalink: permalink, PathID: int64(len(path)),
			SubgraphNames: names, Subgraphs: sgs,
			InLinks: map[string]struct{}{},
			Assets:  map[string]struct{}{},
		}
	}

	notes := map[string]*model.NoteView{
		"target": mk("target.md", "/target", nil),
		"open":   mk("open.md", "/open", map[string]*model.NoteSubgraph{"team": {Name: "team"}}),
		"closed": mk("closed.md", "/closed", map[string]*model.NoteSubgraph{"vault": {Name: "vault"}}),
		"teased": mk("teased.md", "/teased", map[string]*model.NoteSubgraph{"shop": {Name: "shop", Teaser: true}}),
	}
	notes["target"].Free = true

	nvs := &model.NoteViews{
		Map:     map[string]*model.NoteView{},
		PathMap: map[string]*model.NoteView{},
		Version: "live",
	}
	for _, nv := range notes {
		nvs.Map[nv.Permalink] = nv
		nvs.PathMap[nv.Path] = nv
		nvs.List = append(nvs.List, nv)
	}

	return nvs, notes
}

// The template layer gets a per-request read predicate. It runs the one access
// mechanism (canreadnote) and looks up the viewer's subgraphs once, so a page
// with many backlinks does not turn into one DB round trip per link.
func TestResolve_AccessPredicate(t *testing.T) {
	nvs, notes := teaserAccessNotes()

	subgraphCalls := 0
	env := &EnvMock{
		ReaderMovesActiveFunc: func() bool { return false },
		LoggerFunc:            func() logger.Logger { return &logger.DummyLogger{} },
		FeaturesFunc:          func() features.Features { return features.Features{} },
		SiteConfigFunc:        func(context.Context) model.SiteConfig { return model.SiteConfig{} },
		SiteTitleTemplateFunc: func() string { return "%s" },
		LiveNoteViewsFunc:     func() *model.NoteViews { return nvs },
		LatestNoteViewsFunc:   func() *model.NoteViews { return nvs },
		CanReadNoteFunc: func(context.Context, *model.NoteView) (bool, error) {
			return true, nil
		},
		ListActiveUserSubgraphsFunc: func(context.Context, int64) ([]string, error) {
			subgraphCalls++
			return []string{"team"}, nil
		},
		RecordUserNoteViewFunc: func(context.Context, int64, *model.NoteView, *int64) {},
		LastUserNoteViewFunc: func(context.Context, db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
			return db.LastUserNoteViewRow{}, nil
		},
		GetTelegramPostLinksByNoteVersionIDFunc: func(
			context.Context,
			db.GetTelegramPostLinksByNoteVersionIDParams,
		) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
			return nil, nil
		},
	}

	resp, err := rendernotepage.Resolve(context.Background(), env, rendernotepage.Request{
		Path:      "/target",
		UserToken: &usertoken.Data{ID: 7, Role: "user"},
	})
	require.NoError(t, err)

	access := resp.Access()
	require.NotNil(t, access, "the template layer must get a viewer predicate")

	subgraphCalls = 0

	require.True(t, access(notes["open"]), "a note in a subgraph the viewer holds is readable")
	require.False(t, access(notes["closed"]), "a note in a subgraph the viewer lacks is not")
	require.False(t, access(notes["teased"]), "the teaser flag is about display, not readability")
	require.False(t, access(notes["closed"]), "a repeated question costs nothing")

	require.Equal(t, 1, subgraphCalls, "the viewer's subgraphs are looked up once per request, not once per note")
}
