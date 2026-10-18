package runnotequery_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/runnotequery"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	appmodel "trip2g/internal/model"
)

func load(t *testing.T, sources map[string]string) *appmodel.NoteViews {
	t.Helper()
	var srcs []mdloader.SourceFile
	for p, c := range sources {
		srcs = append(srcs, mdloader.SourceFile{Path: p, Content: []byte(c)})
	}
	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	require.NoError(t, err)
	return nvs
}

func query(src string) string {
	return "---\ntype: note-query\n---\n```jsonnet\n" + src + "\n```\n"
}

func TestResolve(t *testing.T) {
	vault := map[string]string{
		"queries/titles.md": query(`[n.title for n in notes if std.startsWith(n.path, 'blog/')]`),
		"queries/by_tag.md": query(`[n.path for n in notes if std.member(std.get(n.meta, 'tags', []), params.tag)]`),
		"queries/broken.md": query(`error 'boom'`),
		"queries/secret.md": query(`[]`),
		"queries/plain.md":  "```jsonnet\n[]\n```\n",
		"blog/a.md":         "---\ntitle: A\ntags: [go]\n---\na",
		"blog/b.md":         "---\ntitle: B\ntags: [web]\n---\nb",
		"blog/paid.md":      "---\ntitle: Paid\ntags: [go]\n---\npaid",
		"queries/paths.md":  query(`[n.path for n in notes if !std.startsWith(n.path, 'queries/')]`),
		"queries/two.md":    query(`[]`) + "```jsonnet\n[]\n```\n",
		"queries/nan.md":    query(`[n.meta.x for n in notes if std.objectHas(n.meta, 'x')]`),
		"_footer.md":        "---\nfree: true\n---\nfooter",
		"board.canvas":      `{"nodes":[],"edges":[]}`,
		"odd.md":            "---\nx: .nan\n---\nodd",
	}
	hidden := map[string]bool{"blog/paid.md": true, "queries/secret.md": true}

	str := func(s string) *string { return &s }

	tests := []struct {
		name     string
		input    model.NoteQueryInput
		expected model.NoteQueryOrErrorPayload
		observed string // "query result" passed to ObserveNoteQuery
	}{
		{
			name:     "returns the query result as JSON",
			input:    model.NoteQueryInput{Query: "queries/titles.md"},
			expected: &model.NoteQueryPayload{JSON: `["A","B"]`},
			observed: "queries/titles.md ok",
		},
		{
			name:     "wikilink target and params",
			input:    model.NoteQueryInput{Query: "[[by_tag]]", Params: str(`{"tag": "go"}`)},
			expected: &model.NoteQueryPayload{JSON: `["blog/a.md"]`},
			observed: "queries/by_tag.md ok",
		},
		{
			name:     "unknown query",
			input:    model.NoteQueryInput{Query: "queries/nope.md"},
			expected: &model.ErrorPayload{Message: "query not found"},
			observed: "unknown not_found",
		},
		{
			name:     "a query the caller can't read looks missing",
			input:    model.NoteQueryInput{Query: "queries/secret.md"},
			expected: &model.ErrorPayload{Message: "query not found"},
			observed: "unknown not_found",
		},
		{
			name:     "a note without type note-query is not a query",
			input:    model.NoteQueryInput{Query: "queries/plain.md"},
			expected: &model.ErrorPayload{Message: "queries/plain.md is not a note-query"},
			observed: "queries/plain.md error",
		},
		{
			name:     "params must be JSON",
			input:    model.NoteQueryInput{Query: "queries/titles.md", Params: str(`{tag`)},
			expected: &model.ErrorPayload{Message: "params must be JSON"},
			observed: "queries/titles.md error",
		},
		{
			name:     "notes skip unreadable, system and non-markdown files",
			input:    model.NoteQueryInput{Query: "queries/paths.md"},
			expected: &model.NoteQueryPayload{JSON: `["blog/a.md","blog/b.md","odd.md"]`},
			observed: "queries/paths.md ok",
		},
		{
			name:     "a non-finite number in frontmatter becomes null",
			input:    model.NoteQueryInput{Query: "queries/nan.md"},
			expected: &model.NoteQueryPayload{JSON: `[null]`},
			observed: "queries/nan.md ok",
		},
		{
			name:     "a query needs exactly one jsonnet block",
			input:    model.NoteQueryInput{Query: "queries/two.md"},
			expected: &model.ErrorPayload{Message: "queries/two.md: a note-query needs exactly one jsonnet block"},
			observed: "queries/two.md error",
		},
		{
			name:     "params are capped",
			input:    model.NoteQueryInput{Query: "queries/titles.md", Params: str(`"` + strings.Repeat("x", 17000) + `"`)},
			expected: &model.ErrorPayload{Message: "params is larger than 16 KB"},
			observed: "queries/titles.md error",
		},
		{
			name:     "runtime error",
			input:    model.NoteQueryInput{Query: "queries/broken.md"},
			expected: &model.ErrorPayload{Message: "queries/broken.md: RUNTIME ERROR: boom"},
			observed: "queries/broken.md error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nvs := load(t, vault)
			var observed []string
			env := &runnotequery.EnvMock{
				ObserveNoteQueryFunc: func(query, result string, _ time.Duration, _ int) {
					observed = append(observed, query+" "+result)
				},
				LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
				CanReadNoteFunc: func(_ context.Context, note *appmodel.NoteView) (bool, error) {
					return !hidden[note.Path], nil
				},
			}

			got, err := runnotequery.Resolve(context.Background(), env, tt.input)
			require.NoError(t, err)
			if diff := pretty.Diff(tt.expected, got); len(diff) > 0 {
				t.Errorf("mismatch:\n%s\ngot: %# v", diff, pretty.Formatter(got))
			}
			require.Equal(t, []string{tt.observed}, observed)
		})
	}
}
