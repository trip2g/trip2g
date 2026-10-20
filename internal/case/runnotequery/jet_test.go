package runnotequery_test

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/runnotequery"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type noteQueryEnv struct {
	nvs  *model.NoteViews
	user *usertoken.Data
}

func (e noteQueryEnv) LatestNoteViews() *model.NoteViews { return e.nvs }
func (e noteQueryEnv) CurrentFederatedScope(context.Context) ([]string, bool) {
	return nil, false
}
func (e noteQueryEnv) ListActiveUserSubgraphs(context.Context, int64) ([]string, error) {
	return nil, nil
}
func (e noteQueryEnv) ObserveNoteQuery(string, string, time.Duration, int) {}

func (e noteQueryEnv) CachedQueryNotes(_ uint64, _ canreadnote.Class, _ bool, build func() ([]byte, error)) ([]byte, error) {
	return build()
}
func (e noteQueryEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return e.user, nil
}

func renderWithNoteQuery(t *testing.T, tmpl string) (string, error) {
	t.Helper()

	srcs := []mdloader.SourceFile{
		{Path: "queries/by_tag.md", Content: []byte("---\ntype: note-query\nfree: true\n---\n```jsonnet\n" +
			"[{title: n.title, tags: n.meta.tags} for n in notes" +
			" if std.member(std.get(n.meta, 'tags', []), params.tag)]\n```\n")},
		{Path: "queries/counts.md", Content: []byte("---\ntype: note-query\nfree: true\n---\n```jsonnet\n" +
			"{posts: std.length([n for n in notes if std.startsWith(n.path, 'blog/')])}\n```\n")},
		{Path: "queries/me.md", Content: []byte("---\ntype: note-query\nfree: true\n---\n```jsonnet\ncurrentUser\n```\n")},
		{Path: "queries/blog.md", Content: []byte("---\ntype: note-query\nfree: true\nall_notes: true\n---\n```jsonnet\n" +
			"[{path: n.path} for n in notes if std.startsWith(n.path, 'blog/')]\n```\n")},
		{Path: "blog/a.md", Content: []byte("---\ntitle: A\nfree: true\ntags: [go]\n---\na")},
		{Path: "blog/b.md", Content: []byte("---\ntitle: B\nfree: true\ntags: [web]\n---\nb")},
		{Path: "blog/paid.md", Content: []byte("---\ntitle: Paid\ntags: [go]\n---\np")},
	}
	log := logger.TestLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	require.NoError(t, err)

	loader := jet.NewInMemLoader()
	loader.Set("page.html", tmpl)
	set := jet.NewSet(loader)
	view, err := set.GetTemplate("page.html")
	require.NoError(t, err)

	vars := make(jet.VarMap)
	user := &usertoken.Data{ID: 7, Role: "editor"}
	vars["noteQuery"] = reflect.ValueOf(runnotequery.JetFunc(context.Background(), noteQueryEnv{nvs: nvs, user: user}))
	vars["noteQueryRows"] = reflect.ValueOf(runnotequery.JetRowsFunc(context.Background(), noteQueryEnv{nvs: nvs, user: user}, ""))

	var out strings.Builder
	err = view.Execute(&out, vars, nil)
	return strings.TrimSpace(out.String()), err
}

func TestNoteQueryInLayout(t *testing.T) {
	got, err := renderWithNoteQuery(t,
		`{{ range _, p := noteQuery("[[by_tag]]", map("tag", "go")) }}{{ p.title }};{{ end }}`)
	require.NoError(t, err)
	require.Equal(t, "A;", got, "the guest doesn't see the paid post")

	got, err = renderWithNoteQuery(t, `{{ c := noteQuery("queries/counts.md") }}{{ c.posts }}`)
	require.NoError(t, err)
	require.Equal(t, "2", got)

	got, err = renderWithNoteQuery(t, `{{ u := noteQuery("queries/me.md") }}{{ u.role }} {{ u.id }}`)
	require.NoError(t, err)
	require.Equal(t, "editor 7", got, "currentUser reaches a template query")

	_, err = renderWithNoteQuery(t, `{{ noteQuery("[[nope]]") }}`)
	require.ErrorContains(t, err, `noteQuery("[[nope]]"): query not found`)
}

func TestNoteQueryRowsInLayout(t *testing.T) {
	got, err := renderWithNoteQuery(t, `{{ range _, row := noteQueryRows("[[blog]]") }}`+
		`{{ row.Data().path }}={{ if p := row.Note(); p }}{{ p.Title() }}{{ else }}nil{{ end }};{{ end }}`)
	require.NoError(t, err)
	require.Equal(t, "blog/a.md=A;blog/b.md=B;blog/paid.md=nil;", got,
		"all_notes gives the program the paid post, but its row has no note for this reader")

	got, err = renderWithNoteQuery(t, `{{ rows := noteQueryRows("[[by_tag]]", map("tag", "go")) }}`+
		`{{ len(rows) }} {{ rows[0].Data().title }}`)
	require.NoError(t, err)
	require.Equal(t, "1 A", got)

	_, err = renderWithNoteQuery(t, `{{ noteQueryRows("queries/counts.md") }}`)
	require.ErrorContains(t, err, `noteQueryRows("queries/counts.md"): the result is not a list`)

	_, err = renderWithNoteQuery(t, `{{ noteQueryRows("[[nope]]") }}`)
	require.ErrorContains(t, err, `noteQueryRows("[[nope]]"): query not found`)
}

// docJetSnippet is the template example on the user pages.
const docJetSnippet = `{{ range _, p := noteQuery("[[by_tag]]", map("tag", "go")) }}
  <a href="{{ p.url }}">{{ p.title }}</a>
{{ end }}`

// docJetRowsSnippet is the rows example on the user pages.
const docJetRowsSnippet = `{{ range _, row := noteQueryRows("[[newest]]") }}
  {{ if post := row.Note(); post }}
    <a href="{{ post.Permalink() }}">{{ post.Title() }}</a>
  {{ end }}
{{ end }}`

func TestDocJetSnippet(t *testing.T) {
	nvs := loadDemoVault(t)

	render := func(tmpl string) string {
		loader := jet.NewInMemLoader()
		loader.Set("page.html", tmpl)
		view, err := jet.NewSet(loader).GetTemplate("page.html")
		require.NoError(t, err)

		vars := make(jet.VarMap)
		vars["noteQuery"] = reflect.ValueOf(runnotequery.JetFunc(context.Background(), noteQueryEnv{nvs: nvs}))
		vars["noteQueryRows"] = reflect.ValueOf(runnotequery.JetRowsFunc(context.Background(), noteQueryEnv{nvs: nvs}, ""))

		var out strings.Builder
		require.NoError(t, view.Execute(&out, vars, nil))
		return strings.Join(strings.Fields(out.String()), " ")
	}

	require.Equal(t,
		`<a href="/blog/go_on_the_web">Go on the web</a> <a href="/blog/hello_go">Hello, Go</a>`,
		render(docJetSnippet))
	require.Equal(t,
		`<a href="/blog/go_on_the_web">Go on the web</a> <a href="/blog/web_basics">Web basics</a> `+
			`<a href="/blog/hello_go">Hello, Go</a>`,
		render(docJetRowsSnippet))

	for _, page := range []string{"en/user/note-queries", "ru/user/note-queries"} {
		doc, readErr := os.ReadFile("../../../docs/" + page + ".md")
		require.NoError(t, readErr)
		require.Contains(t, string(doc), "```jet\n"+docJetSnippet+"\n```", "docs/%s.md lost the template example", page)
		require.Contains(t, string(doc), "```jet\n"+docJetRowsSnippet+"\n```", "docs/%s.md lost the rows example", page)
	}
}
