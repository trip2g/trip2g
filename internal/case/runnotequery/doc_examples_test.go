package runnotequery_test

import (
	"context"
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
)

const demoVault = "testdata/vault"

// TestDocExamples runs the demo queries from the user docs as a guest (only
// free notes are readable) and as an admin.
func TestDocExamples(t *testing.T) {
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
		srcs = append(srcs, mdloader.SourceFile{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	require.NoError(t, err)

	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	require.NoError(t, err)

	run := func(admin bool, query, params string) string {
		env := &runnotequery.EnvMock{
			LatestNoteViewsFunc:  func() *appmodel.NoteViews { return nvs },
			ObserveNoteQueryFunc: func(string, string, time.Duration, int) {},
			CanReadNoteFunc: func(_ context.Context, note *appmodel.NoteView) (bool, error) {
				return admin || note.Free, nil
			},
		}
		input := model.NoteQueryInput{Query: query}
		if params != "" {
			input.Params = &params
		}
		res, resErr := runnotequery.Resolve(context.Background(), env, input)
		require.NoError(t, resErr)
		payload, ok := res.(*model.NoteQueryPayload)
		require.True(t, ok, "%s: %#v", query, res)
		return payload.JSON
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
	}
	for _, tt := range tests {
		require.JSONEq(t, tt.expected, run(tt.admin, tt.query, tt.params), "%s %s admin=%v", tt.query, tt.params, tt.admin)
	}
}

// TestDocSnippets keeps the examples on the user pages identical to the demo
// queries that TestDocExamples runs.
func TestDocSnippets(t *testing.T) {
	for _, page := range []string{"en/user/note-queries", "ru/user/note-queries"} {
		doc, err := os.ReadFile("../../../docs/" + page + ".md")
		require.NoError(t, err)

		for _, name := range []string{"by_tag", "latest", "task_counts"} {
			src, readErr := os.ReadFile(demoVault + "/queries/" + name + ".md")
			require.NoError(t, readErr)

			_, block, ok := strings.Cut(string(src), "```jsonnet\n")
			require.True(t, ok, name)
			block, _, _ = strings.Cut(block, "\n```")

			require.Contains(t, string(doc), "```jsonnet\n"+block+"\n```", "docs/%s.md lost the %s example", page, name)
		}
	}
}
