package runnotequery_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

const demoVault = "testdata/vault"

// loadDemoVault loads the demo vault with ids, as a synced vault has them.
func loadDemoVault(t *testing.T) *appmodel.NoteViews {
	t.Helper()
	var srcs []mdloader.SourceFile
	err := filepath.WalkDir(demoVault, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(demoVault, p)
		id := int64(len(srcs) + 1)
		srcs = append(srcs, mdloader.SourceFile{Path: filepath.ToSlash(rel), PathID: id, VersionID: 100 + id, Content: content})
		return nil
	})
	require.NoError(t, err)

	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	require.NoError(t, err)
	return nvs
}

// TestDocExamples runs the demo queries from the user docs as a guest (only
// free notes are readable) and as an admin.
func TestDocExamples(t *testing.T) {
	nvs := loadDemoVault(t)

	run := func(admin bool, query, params string) string {
		env := &runnotequery.EnvMock{
			LatestNoteViewsFunc:       func() *appmodel.NoteViews { return nvs },
			CachedQueryNotesFunc:      runnotequery.NewNotesCache().CachedQueryNotes,
			ObserveNoteQueryFunc:      func(string, string, time.Duration, int) {},
			CurrentFederatedScopeFunc: func(context.Context) ([]string, bool) { return nil, false },
			CurrentUserTokenFunc: func(context.Context) (*usertoken.Data, error) {
				if admin {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				}
				return nil, nil
			},
		}
		input := model.NoteQueryInput{Query: query}
		if params != "" {
			var v any
			require.NoError(t, json.Unmarshal([]byte(params), &v))
			input.Params = v
		}
		res, resErr := runnotequery.Resolve(context.Background(), env, input)
		require.NoError(t, resErr)
		payload, ok := res.(*model.NoteQueryPayload)
		require.True(t, ok, "%s: %#v", query, res)
		raw, ok := payload.Result.(json.RawMessage)
		require.True(t, ok, "%s: %T", query, payload.Result)
		return string(raw)
	}

	tests := []struct {
		admin         bool
		query, params string
		expected      string
	}{
		{false, "[[by_tag]]", `{"tag": "go"}`,
			`[{"date":"2026-10-10","title":"Go on the web","url":"/blog/go_on_the_web"},` +
				`{"date":"2026-10-01","title":"Hello, Go","url":"/blog/hello_go"}]`},
		{true, "[[by_tag]]", `{"tag": "go"}`,
			`[{"date":"2026-10-12","title":"Members only","url":"/blog/members_only"},` +
				`{"date":"2026-10-10","title":"Go on the web","url":"/blog/go_on_the_web"},` +
				`{"date":"2026-10-01","title":"Hello, Go","url":"/blog/hello_go"}]`},
		{false, "[[latest]]", `{"limit": 2}`, `["Go on the web","Web basics"]`},
		{false, "[[latest]]", ``, `["Go on the web","Web basics","Hello, Go"]`},
		{false, "[[task_counts]]", ``, `{"done":1,"todo":2}`},
		{false, "[[folder_counts]]", ``, `{"blog":3,"queries":6,"tasks":3}`},
		{true, "[[folder_counts]]", ``, `{"blog":4,"queries":6,"tasks":3}`},
		{false, "[[teaser]]", ``, `{"locked":1,"open":["Go on the web","Hello, Go","Web basics"],"signedIn":false}`},
		{true, "[[teaser]]", ``, `{"locked":0,"open":["Go on the web","Hello, Go","Members only","Web basics"],"signedIn":true}`},
	}
	for _, tt := range tests {
		require.JSONEq(t, tt.expected, run(tt.admin, tt.query, tt.params), "%s %s admin=%v", tt.query, tt.params, tt.admin)
	}
}

// TestDocRowsExample resolves the rows of the `newest` example the way
// `rows { note { title } }` does.
func TestDocRowsExample(t *testing.T) {
	nvs := loadDemoVault(t)

	titles := func(user *usertoken.Data) []string {
		env := &runnotequery.EnvMock{
			LatestNoteViewsFunc:       func() *appmodel.NoteViews { return nvs },
			CachedQueryNotesFunc:      runnotequery.NewNotesCache().CachedQueryNotes,
			ObserveNoteQueryFunc:      func(string, string, time.Duration, int) {},
			CurrentFederatedScopeFunc: func(context.Context) ([]string, bool) { return nil, false },
			CurrentUserTokenFunc:      func(context.Context) (*usertoken.Data, error) { return user, nil },
		}
		res, err := runnotequery.Resolve(context.Background(), env, model.NoteQueryInput{Query: "[[newest]]"})
		require.NoError(t, err)
		payload, ok := res.(*model.NoteQueryPayload)
		require.True(t, ok, "%#v", res)

		rows, err := runnotequery.Rows(payload)
		require.NoError(t, err)
		var out []string
		for i := range rows {
			note, noteErr := runnotequery.RowNote(&rows[i])
			require.NoError(t, noteErr)
			require.NotNil(t, note)
			out = append(out, note.Title)
		}
		return out
	}

	require.Equal(t, []string{"Go on the web", "Web basics", "Hello, Go"}, titles(nil))
	require.Equal(t, []string{"Members only", "Go on the web", "Web basics"}, titles(&usertoken.Data{ID: 1, Role: "admin"}))
}

// TestDocSnippets keeps the examples on the user pages identical to the demo
// queries that TestDocExamples runs.
func TestDocSnippets(t *testing.T) {
	for _, page := range []string{"en/user/note-queries", "ru/user/note-queries"} {
		doc, err := os.ReadFile("../../../docs/" + page + ".md")
		require.NoError(t, err)

		for _, name := range []string{"by_tag", "latest", "task_counts", "folder_counts", "teaser", "newest"} {
			src, readErr := os.ReadFile(demoVault + "/queries/" + name + ".md")
			require.NoError(t, readErr)

			_, block, ok := strings.Cut(string(src), "```jsonnet\n")
			require.True(t, ok, name)
			block, _, _ = strings.Cut(block, "\n```")

			require.Contains(t, string(doc), "```jsonnet\n"+block+"\n```", "docs/%s.md lost the %s example", page, name)
		}
	}
}
