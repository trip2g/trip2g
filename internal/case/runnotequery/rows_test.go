package runnotequery_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/runnotequery"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

// rowsVault has notes with ids, as a synced vault does.
func rowsVault(t *testing.T) *appmodel.NoteViews {
	t.Helper()
	sources := []mdloader.SourceFile{
		{Path: "blog/a.md", PathID: 1, VersionID: 11, Content: []byte("---\ntitle: A\nfree: true\n---\na")},
		{Path: "blog/b.md", PathID: 2, VersionID: 12, Content: []byte("---\ntitle: B\nfree: true\nsearch: false\n---\nb")},
		{Path: "blog/paid.md", PathID: 3, VersionID: 13, Content: []byte("---\ntitle: Paid\n---\npaid")},
		{Path: "_footer.md", PathID: 4, VersionID: 14, Content: []byte("---\nfree: true\n---\nfooter")},
		{Path: "queries/ids.md", PathID: 5, VersionID: 15, Content: []byte(query(
			`[{path: n.path, path_id: n.path_id, version_id: n.version_id, search: std.get(n.meta, 'search', true)}` +
				` for n in notes if std.startsWith(n.path, 'blog/')]`))},
		{Path: "queries/echo.md", PathID: 6, VersionID: 16, Content: []byte(query(`params.rows`))},
		{Path: "queries/all.md", PathID: 7, VersionID: 17, Content: []byte("---\ntype: note-query\nfree: true\nall_notes: true\n---\n" +
			"```jsonnet\n[{version_id: n.version_id} for n in notes if std.startsWith(n.path, 'blog/')]\n```\n")},
	}
	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: sources, Log: &log})
	require.NoError(t, err)
	return nvs
}

func rowsEnv(nvs *appmodel.NoteViews, user *usertoken.Data) *runnotequery.EnvMock {
	return &runnotequery.EnvMock{
		LatestNoteViewsFunc:       func() *appmodel.NoteViews { return nvs },
		ObserveNoteQueryFunc:      func(string, string, time.Duration, int) {},
		CurrentFederatedScopeFunc: func(context.Context) ([]string, bool) { return nil, false },
		CurrentUserTokenFunc:      func(context.Context) (*usertoken.Data, error) { return user, nil },
		ListActiveUserSubgraphsFunc: func(context.Context, int64) ([]string, error) {
			return nil, nil
		},
	}
}

func runRows(t *testing.T, env runnotequery.Env, input model.NoteQueryInput) *model.NoteQueryPayload {
	t.Helper()
	res, err := runnotequery.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	payload, ok := res.(*model.NoteQueryPayload)
	require.True(t, ok, "%#v", res)
	return payload
}

// rowNotePaths resolves every row's note and returns its permalink, or "" for
// a row without one.
func rowNotePaths(t *testing.T, payload *model.NoteQueryPayload) []string {
	t.Helper()
	rows, err := runnotequery.Rows(payload)
	require.NoError(t, err)

	paths := make([]string, len(rows))
	for i := range rows {
		note, noteErr := runnotequery.RowNote(&rows[i])
		require.NoError(t, noteErr)
		if note != nil {
			paths[i] = note.Path
		}
	}
	return paths
}

func TestQueryNotesCarryIDsAndSearchFalseNotes(t *testing.T) {
	nvs := rowsVault(t)
	payload := runRows(t, rowsEnv(nvs, nil), model.NoteQueryInput{Query: "queries/ids.md"})

	require.JSONEq(t, `[
		{"path":"blog/a.md","path_id":1,"version_id":11,"search":true},
		{"path":"blog/b.md","path_id":2,"version_id":12,"search":false}
	]`, string(payload.Result.(json.RawMessage)))
}

func TestRowNotes(t *testing.T) {
	nvs := rowsVault(t)

	tests := []struct {
		name     string
		rows     any
		expected []string
	}{
		{
			name:     "version_id, path_id and path each name a note",
			rows:     []any{map[string]any{"version_id": 11}, map[string]any{"path_id": 2}, map[string]any{"path": "blog/a.md"}},
			expected: []string{"/blog/a", "/blog/b", "/blog/a"},
		},
		{
			name: "version_id wins over path_id, path_id over path",
			rows: []any{
				map[string]any{"version_id": 12, "path_id": 1, "path": "blog/a.md"},
				map[string]any{"path_id": 2, "path": "blog/a.md"},
			},
			expected: []string{"/blog/b", "/blog/b"},
		},
		{
			name: "an unknown id does not fall back to the next key",
			rows: []any{
				map[string]any{"version_id": 999, "path_id": 1},
				map[string]any{"path_id": 999, "path": "blog/a.md"},
				map[string]any{"path": "blog/nope.md"},
			},
			expected: []string{"", "", ""},
		},
		{
			name:     "an item without a reference, or not an object, has no note",
			rows:     []any{map[string]any{"title": "A"}, "blog/a.md", 11, nil, map[string]any{"version_id": "11"}},
			expected: []string{"", "", "", "", ""},
		},
		{
			name:     "a note the viewer can't read is null",
			rows:     []any{map[string]any{"version_id": 13}, map[string]any{"path": "blog/paid.md"}},
			expected: []string{"", ""},
		},
		{
			name:     "system notes stay out, as in notes",
			rows:     []any{map[string]any{"path": "_footer.md"}, map[string]any{"version_id": 14}},
			expected: []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := model.NoteQueryInput{Query: "queries/echo.md", Params: map[string]any{"rows": tt.rows}}
			payload := runRows(t, rowsEnv(nvs, nil), input)
			require.Equal(t, tt.expected, rowNotePaths(t, payload))
		})
	}
}

func TestRowNotesRecheckAccessForAllNotes(t *testing.T) {
	nvs := rowsVault(t)

	guest := runRows(t, rowsEnv(nvs, nil), model.NoteQueryInput{Query: "queries/all.md"})
	require.JSONEq(t, `[{"version_id":11},{"version_id":12},{"version_id":13}]`,
		string(guest.Result.(json.RawMessage)), "all_notes hands the program the paid note")
	require.Equal(t, []string{"/blog/a", "/blog/b", ""}, rowNotePaths(t, guest),
		"the row still doesn't resolve a note the guest can't read")

	admin := runRows(t, rowsEnv(nvs, &usertoken.Data{ID: 1, Role: "admin"}), model.NoteQueryInput{Query: "queries/all.md"})
	require.Equal(t, []string{"/blog/a", "/blog/b", "/blog/paid"}, rowNotePaths(t, admin))
}

func TestRowsOnlyForListResults(t *testing.T) {
	nvs := rowsVault(t)
	env := rowsEnv(nvs, nil)

	for _, result := range []any{map[string]any{"a": 1}, 3, "x", nil} {
		payload := runRows(t, env, model.NoteQueryInput{Query: "queries/echo.md", Params: map[string]any{"rows": result}})
		rows, err := runnotequery.Rows(payload)
		require.NoError(t, err)
		require.Nil(t, rows, "%v", result)
	}

	payload := runRows(t, env, model.NoteQueryInput{Query: "queries/echo.md", Params: map[string]any{"rows": []any{}}})
	rows, err := runnotequery.Rows(payload)
	require.NoError(t, err)
	require.Equal(t, []model.NoteQueryRow{}, rows)

	payload = runRows(t, env, model.NoteQueryInput{Query: "queries/echo.md", Params: map[string]any{"rows": []any{map[string]any{"x": 1.5}}}})
	rows, err = runnotequery.Rows(payload)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	data, ok := rows[0].Data.(json.RawMessage)
	require.True(t, ok)
	require.JSONEq(t, `{"x":1.5}`, string(data))
}

func TestRowNotesAreLazy(t *testing.T) {
	nvs := rowsVault(t)
	env := rowsEnv(nvs, &usertoken.Data{ID: 5, Role: "user"})
	input := model.NoteQueryInput{Query: "queries/echo.md", Params: map[string]any{
		"rows": []any{map[string]any{"version_id": 11}, map[string]any{"version_id": 12}},
	}}

	payload := runRows(t, env, input)
	require.Len(t, env.ListActiveUserSubgraphsCalls(), 1, "the query itself")

	rows, err := runnotequery.Rows(payload)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Len(t, env.ListActiveUserSubgraphsCalls(), 1, "rows without note look nothing up")

	require.Equal(t, []string{"/blog/a", "/blog/b"}, rowNotePaths(t, payload))
	require.Len(t, env.ListActiveUserSubgraphsCalls(), 2, "the viewer is resolved once for all rows")
}
