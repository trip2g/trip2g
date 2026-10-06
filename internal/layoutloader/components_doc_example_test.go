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

// The worked example in docs/{en,ru}/user/components.md; TestComponentsDocSnippetsInDocs keeps them in sync.
const componentsDocCard = `{{ block _style_@lid() }}
.@did {
  display: block;
  padding: 16px;
  border: 1px solid #ddd;
  border-radius: 8px;
}
.@did--featured { border-color: #0070f3; }
.@did__title { margin: 0 0 8px; font-size: 1.25rem; }
{{ end }}

{{ block @lid(title="", url="", featured=false) }}
<a class="@did{{ if featured }} @did--featured{{ end }}" href="{{ url }}">
  <h3 class="@did__title">{{ title }}</h3>
  {{ yield content }}
</a>
{{ end }}`

const componentsDocButton = `{{ block _style_@lid() }}
.@did {
  display: inline-block;
  padding: 8px 16px;
  border-radius: 6px;
  background: #0070f3;
  color: #fff;
}
{{ end }}

{{ block @lid(label="", url="") }}
<a class="@did" href="{{ url }}">{{ label }}</a>
{{ end }}`

const componentsDocPage = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ note.Title() }}</title>
  <style>{{ yield_blocks("_style_") }}</style>
</head>
<body>
  {{ note.HTMLString() }}

  {{ blog := nvs.ByGlob("blog/*.md").Public() }}
  {{ sorted := blog.SortByMeta("date").Desc().SortBy("Title") }}
  {{ range i, post := sorted.Limit(3).All() }}
    {{ yield components_card(
      title=post.Title(),
      url=post.PermalinkEncoded(),
      featured=i == 0
    ) content }}
      {{ if summary := post.M().GetString("summary", ""); summary }}
        <p>{{ summary }}</p>
      {{ end }}
    {{ end }}
  {{ end }}

  {{ yield components_button(label="All posts", url="/blog") }}
</body>
</html>`

func TestComponentsDocExample(t *testing.T) {
	notes := model.NewNoteViews()
	add := func(path, title string, free bool, meta map[string]interface{}) {
		notes.PathMap[path] = &model.NoteView{
			Path:      path,
			Title:     title,
			Permalink: "/" + strings.TrimSuffix(path, ".md"),
			Free:      free,
			RawMeta:   meta,
		}
	}
	add("blog/a.md", "Alpha", true, map[string]interface{}{"date": "2024-03-01", "summary": "First & best"})
	add("blog/b.md", "Beta", true, map[string]interface{}{"date": "2024-05-10"})
	add("blog/c.md", "Gamma", true, map[string]interface{}{"date": "2024-01-15"})
	add("blog/d.md", "Delta", true, map[string]interface{}{"date": "2023-12-31"})
	add("blog/paid.md", "Paid", false, map[string]interface{}{"date": "2025-01-01"})

	sources := []model.LayoutSourceFile{
		{ID: "/page", Path: "_layouts/page.html", Content: componentsDocPage},
		{ID: "/components/card", Path: "_layouts/components/card.html", Content: componentsDocCard},
		{ID: "/components/button", Path: "_layouts/components/button.html", Content: componentsDocButton},
	}
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)
	require.Empty(t, layouts.Map["/page"].Warnings)

	page := &model.NoteView{Path: "index.md", Title: "Home", HTML: "<p>Hello</p>"}
	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(page))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(notes, "live"))

	var buf bytes.Buffer
	err = layouts.Map["/page"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	want := `<!DOCTYPE html> <html lang="en"> <head> <meta charset="utf-8"> <title>Home</title> <style>` +
		` .components-card { display: block; padding: 16px; border: 1px solid #ddd; border-radius: 8px; }` +
		` .components-card--featured { border-color: #0070f3; }` +
		` .components-card__title { margin: 0 0 8px; font-size: 1.25rem; }` +
		` .components-button { display: inline-block; padding: 8px 16px; border-radius: 6px; background: #0070f3; color: #fff; }` +
		` </style> </head> <body> <p>Hello</p>` +
		` <a class="components-card components-card--featured" href="/blog/b"> <h3 class="components-card__title">Beta</h3> </a>` +
		` <a class="components-card" href="/blog/a"> <h3 class="components-card__title">Alpha</h3> <p>First &amp; best</p> </a>` +
		` <a class="components-card" href="/blog/c"> <h3 class="components-card__title">Gamma</h3> </a>` +
		` <a class="components-button" href="/blog">All posts</a> </body> </html>`
	require.Equal(t, want, strings.Join(strings.Fields(buf.String()), " "))
}

// The base layer example in docs/{en,ru}/user/components.md.
const componentsDocBase = `{{ block _style_@lid() }}
:root { --text: #1a1a1a; --accent: #0070f3; }
*, *::before, *::after { box-sizing: border-box; }
body { margin: 0; font: 16px/1.5 system-ui, sans-serif; color: var(--text); }
a { color: var(--accent); }
{{ end }}

{{ block @lid(title="") }}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
  <style>{{ yield_blocks("_style_") }}</style>
</head>
<body>
  {{ yield content }}
</body>
</html>
{{ end }}`

const componentsDocBasePage = `{{ yield components_base(title=note.Title()) content }}
  {{ note.HTMLString() }}
  {{ yield components_button(label="All posts", url="/blog") }}
{{ end }}`

func TestComponentsDocBaseExample(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/page", Path: "_layouts/page.html", Content: componentsDocBasePage},
		{ID: "/components/base", Path: "_layouts/components/base.html", Content: componentsDocBase},
		{ID: "/components/button", Path: "_layouts/components/button.html", Content: componentsDocButton},
	}
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)
	require.Empty(t, layouts.Map["/page"].Warnings)

	page := &model.NoteView{Path: "index.md", Title: "Q&A", HTML: "<p>Hello</p>"}
	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(page))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(model.NewNoteViews(), "live"))

	var buf bytes.Buffer
	err = layouts.Map["/page"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	want := `<!DOCTYPE html> <html lang="en"> <head> <meta charset="utf-8"> <title>Q&amp;A</title> <style>` +
		` :root { --text: #1a1a1a; --accent: #0070f3; }` +
		` *, *::before, *::after { box-sizing: border-box; }` +
		` body { margin: 0; font: 16px/1.5 system-ui, sans-serif; color: var(--text); }` +
		` a { color: var(--accent); }` +
		` .components-button { display: inline-block; padding: 8px 16px; border-radius: 6px; background: #0070f3; color: #fff; }` +
		` </style> </head> <body> <p>Hello</p>` +
		` <a class="components-button" href="/blog">All posts</a> </body> </html>`
	require.Equal(t, want, strings.Join(strings.Fields(buf.String()), " "))
}

func TestComponentsDocStyleOrder(t *testing.T) {
	component := func(name, inner string) string {
		return `{{ block _style_@lid() }}/*` + name + `*/{{ end }}{{ block @lid() }}` + inner + `{{ end }}`
	}
	sources := []model.LayoutSourceFile{
		{ID: "/page", Path: "_layouts/page.html", Content: `<style>{{ yield_blocks("_style_") }}</style>{{ yield base() }}{{ yield card() }}`},
		{ID: "/base", Path: "_layouts/base.html", Content: component("base", `{{ yield header() }}`)},
		{ID: "/card", Path: "_layouts/card.html", Content: component("card", `{{ yield button() }}`)},
		{ID: "/header", Path: "_layouts/header.html", Content: component("header", "")},
		{ID: "/button", Path: "_layouts/button.html", Content: component("button", "")},
	}
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)

	var buf bytes.Buffer
	err = layouts.Map["/page"].View.Execute(&buf, make(jet.VarMap), nil)
	require.NoError(t, err)

	require.Equal(t, "<style>/*header*//*button*//*base*//*card*/</style>", strings.TrimSpace(buf.String()))
}

func TestComponentsDocSnippetsInDocs(t *testing.T) {
	snippets := []docSnippet{
		{name: "card", src: componentsDocCard},
		{name: "button", src: componentsDocButton},
		{name: "page", src: componentsDocPage},
		{name: "base", src: componentsDocBase},
		{name: "base page", src: componentsDocBasePage},
	}
	requireSnippetsInDocs(t, []string{"en/user/components", "ru/user/components"}, snippets)
}
