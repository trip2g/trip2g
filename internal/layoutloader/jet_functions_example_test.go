package layoutloader

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

// Mirrors the worked example in docs/{en,ru}/user/jet-functions.md; keep them in sync.
const jetFunctionsExample = `{{ query := nvs.ByGlob("blog/*.md").Public() }}
{{ total := len(query.All()) }}
{{ posts := query.SortByMeta("date").Desc().SortBy("Title").Limit(5).All() }}
<h2>Latest posts</h2>
<p>Showing {{ len(posts) }} of {{ total }}</p>
<ul>
{{ range i, post := posts }}
  <li>
    <a href="{{ post.PermalinkEncoded() }}">{{ post.Title() | html }}</a>
    <time>{{ post.M().GetString("date", "undated") }}</time>
  </li>
{{ else }}
  <li>No posts yet.</li>
{{ end }}
</ul>`

func TestJetFunctionsDocExample(t *testing.T) {
	notes := model.NewNoteViews()
	add := func(path, title string, free bool, date string) {
		meta := map[string]interface{}{}
		if date != "" {
			meta["date"] = date
		}
		notes.PathMap[path] = &model.NoteView{
			Path:      path,
			Title:     title,
			Permalink: "/" + strings.TrimSuffix(path, ".md"),
			Free:      free,
			RawMeta:   meta,
		}
	}
	add("blog/a.md", "Alpha", true, "2024-03-01")
	add("blog/b.md", "Beta", true, "2024-05-10")
	add("blog/c.md", "Gamma & Co", true, "2024-05-10")
	add("blog/d.md", "Delta", true, "2023-12-31")
	add("blog/e.md", "Epsilon", true, "")
	add("blog/f.md", "Zeta", true, "2024-01-15")
	add("blog/paid.md", "Paid", false, "2025-01-01")
	add("blog/_draft.md", "Draft", true, "2025-02-01")
	add("blog/archive/old.md", "Old", true, "2025-03-01")

	sources := []model.LayoutSourceFile{{
		ID:      "/listing",
		Path:    "_layouts/listing.html",
		Content: jetFunctionsExample,
	}}
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(notes, "live"))

	var buf bytes.Buffer
	err = layouts.Map["/listing"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	want := `<h2>Latest posts</h2> <p>Showing 5 of 6</p> <ul> ` +
		`<li> <a href="/blog/b">Beta</a> <time>2024-05-10</time> </li> ` +
		`<li> <a href="/blog/c">Gamma &amp; Co</a> <time>2024-05-10</time> </li> ` +
		`<li> <a href="/blog/a">Alpha</a> <time>2024-03-01</time> </li> ` +
		`<li> <a href="/blog/f">Zeta</a> <time>2024-01-15</time> </li> ` +
		`<li> <a href="/blog/d">Delta</a> <time>2023-12-31</time> </li> ` +
		`</ul>`
	require.Equal(t, want, strings.Join(strings.Fields(buf.String()), " "))
}
