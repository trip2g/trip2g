package runnotequery_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/runnotequery"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type noteQueryEnv struct{ nvs *model.NoteViews }

func (e noteQueryEnv) LatestNoteViews() *model.NoteViews { return e.nvs }
func (e noteQueryEnv) CanReadNote(_ context.Context, n *model.NoteView) (bool, error) {
	return n.Free, nil
}
func (e noteQueryEnv) ObserveNoteQuery(string, string, time.Duration, int) {}
func (e noteQueryEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return nil, nil
}

func renderWithNoteQuery(t *testing.T, tmpl string) (string, error) {
	t.Helper()

	srcs := []mdloader.SourceFile{
		{Path: "queries/by_tag.md", Content: []byte("---\ntype: note-query\nfree: true\n---\n```jsonnet\n" +
			"[{title: n.title, tags: n.meta.tags} for n in notes" +
			" if std.member(std.get(n.meta, 'tags', []), params.tag)]\n```\n")},
		{Path: "queries/counts.md", Content: []byte("---\ntype: note-query\nfree: true\n---\n```jsonnet\n" +
			"{posts: std.length([n for n in notes if std.startsWith(n.path, 'blog/')])}\n```\n")},
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
	vars["noteQuery"] = reflect.ValueOf(runnotequery.JetFunc(context.Background(), noteQueryEnv{nvs: nvs}))

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

	_, err = renderWithNoteQuery(t, `{{ noteQuery("[[nope]]") }}`)
	require.ErrorContains(t, err, `noteQuery("[[nope]]"): query not found`)
}

// docJetSnippet is the template example on the user pages.
const docJetSnippet = `{{ range _, p := noteQuery("[[by_tag]]", map("tag", "go")) }}
  <a href="{{ p.url }}">{{ p.title }}</a>
{{ end }}`

func TestDocJetSnippet(t *testing.T) {
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

	loader := jet.NewInMemLoader()
	loader.Set("page.html", docJetSnippet)
	view, err := jet.NewSet(loader).GetTemplate("page.html")
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["noteQuery"] = reflect.ValueOf(runnotequery.JetFunc(context.Background(), noteQueryEnv{nvs: nvs}))

	var out strings.Builder
	require.NoError(t, view.Execute(&out, vars, nil))
	require.Equal(t,
		`<a href="/blog/go_on_the_web">Go on the web</a> <a href="/blog/hello_go">Hello, Go</a>`,
		strings.Join(strings.Fields(out.String()), " "))

	for _, page := range []string{"en/user/note-queries", "ru/user/note-queries"} {
		doc, readErr := os.ReadFile("../../../docs/" + page + ".md")
		require.NoError(t, readErr)
		require.Contains(t, string(doc), "```jet\n"+docJetSnippet+"\n```", "docs/%s.md lost the template example", page)
	}
}
