package runnotequery_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/runnotequery"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

func TestResolveNotesCacheKeepsAccessClassesApart(t *testing.T) {
	nvs := load(t, map[string]string{
		"queries/titles.md": query(`[n.title for n in notes if !std.startsWith(n.path, 'queries/')]`),
		"queries/all.md": "---\ntype: note-query\nfree: true\nall_notes: true\n---\n```jsonnet\n" +
			"[[n.title, n.readable] for n in notes if !std.startsWith(n.path, 'queries/')]\n```\n",
		"free.md":    "---\ntitle: Free\nfree: true\n---\nf",
		"premium.md": "---\ntitle: Premium\nsubgraph: premium\n---\np",
	})
	nvs.Generation = 1

	cache := runnotequery.NewNotesCache()
	builds := 0
	var viewer *usertoken.Data
	var subgraphs []string
	env := &runnotequery.EnvMock{
		LatestNoteViewsFunc:  func() *appmodel.NoteViews { return nvs },
		ObserveNoteQueryFunc: func(string, string, time.Duration, int) {},
		CurrentUserTokenFunc: func(context.Context) (*usertoken.Data, error) {
			return viewer, nil
		},
		CurrentFederatedScopeFunc: func(context.Context) ([]string, bool) { return nil, false },
		ListActiveUserSubgraphsFunc: func(context.Context, int64) ([]string, error) {
			return subgraphs, nil
		},
		CachedQueryNotesFunc: func(generation uint64, class canreadnote.Class, all bool, build func() ([]byte, error)) ([]byte, error) {
			return cache.CachedQueryNotes(generation, class, all, func() ([]byte, error) {
				builds++
				return build()
			})
		},
	}

	run := func(q string) string {
		t.Helper()
		res, err := runnotequery.Resolve(context.Background(), env, model.NoteQueryInput{Query: q})
		require.NoError(t, err)
		payload, ok := res.(*model.NoteQueryPayload)
		require.True(t, ok, "%#v", res)
		raw, ok := payload.Result.(json.RawMessage)
		require.True(t, ok)
		return string(raw)
	}

	admin := &usertoken.Data{ID: 1, Role: "admin"}
	user := &usertoken.Data{ID: 2, Role: "user"}

	viewer = admin
	require.JSONEq(t, `["Free","Premium"]`, run("queries/titles.md"))
	viewer = nil
	require.JSONEq(t, `["Free"]`, run("queries/titles.md"))
	require.JSONEq(t, `[["Free",true],["Premium",false]]`, run("queries/all.md"))
	viewer, subgraphs = user, nil
	require.JSONEq(t, `["Free"]`, run("queries/titles.md"))
	viewer, subgraphs = user, []string{"premium"}
	require.JSONEq(t, `["Free","Premium"]`, run("queries/titles.md"))
	require.Equal(t, 5, builds)

	viewer = admin
	require.JSONEq(t, `["Free","Premium"]`, run("queries/titles.md"))
	viewer = nil
	require.JSONEq(t, `["Free"]`, run("queries/titles.md"))
	require.Equal(t, 5, builds, "a second call per class is served from the cache")

	next := load(t, map[string]string{
		"queries/titles.md": query(`[n.title for n in notes if !std.startsWith(n.path, 'queries/')]`),
		"free.md":           "---\ntitle: Free again\nfree: true\n---\nf",
	})
	next.Generation = 2
	nvs = next
	require.JSONEq(t, `["Free again"]`, run("queries/titles.md"))
	require.Equal(t, 6, builds, "a new generation rebuilds")
}

func TestNotesCacheIsBounded(t *testing.T) {
	cache := runnotequery.NewNotesCache()
	builds := 0
	build := func() ([]byte, error) {
		builds++
		return []byte(`[]`), nil
	}

	class := func(i int) canreadnote.Class {
		return canreadnote.Class("user:" + strconv.Itoa(i))
	}
	for i := range runnotequery.MaxCachedNotes + 1 {
		_, err := cache.CachedQueryNotes(1, class(i), false, build)
		require.NoError(t, err)
	}
	_, err := cache.CachedQueryNotes(1, class(0), false, build)
	require.NoError(t, err)
	require.Equal(t, runnotequery.MaxCachedNotes+2, builds, "a full cache starts over")

	builds = 0
	_, err = cache.CachedQueryNotes(0, "guest", false, build)
	require.NoError(t, err)
	_, err = cache.CachedQueryNotes(0, "guest", false, build)
	require.NoError(t, err)
	require.Equal(t, 2, builds, "an older generation is never cached")
}
