package layoutloader

import (
	"reflect"
	"strings"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

// The layouts below are copied verbatim into docs/{en,ru}/user/templates.md;
// TestTemplatesDocSnippetsInDocs fails when a page drifts.
const (
	docHTMLInjections = `<head>
  <title>{{ title }}</title>
  {{ range i, injection := htmlInjectionsHead }}
    {{ injection.Content | unsafe }}
  {{ end }}
</head>
<body>
  {{ note.HTMLString() }}
  {{ range i, injection := htmlInjectionsBodyEnd }}
    {{ injection.Content | unsafe }}
  {{ end }}
</body>`

	docSeo = `<head>
  {{ if note.M().GetBool("noindex", false) }}
    <meta name="robots" content="noindex">
  {{ end }}
  {{ canonicalRoute := note.M().GetString("route", "") }}
  {{ if canonicalRoute != "" }}
  <link rel="canonical" href="https://{{ canonicalRoute }}">
  {{ else }}
  <link rel="canonical" href="{{ publicURL }}{{ note.Permalink() }}">
  {{ end }}
</head>`

	docToc = `{{ pr := note.PartialRenderer() }}
<nav class="toc">
  {{ range i, h := note.TOC() }}
    <a class="toc__item toc__item--{{ h.Level }}" href="#{{ h.ID }}">
      {{ h.Text }}
    </a>
  {{ end }}
</nav>

{{ range i, s := pr.Sections(2) }}
  <section id="{{ s.ID }}">
    <h2>{{ s.TitleHTML }}</h2>
    {{ s.ContentHTML }}
  </section>
{{ end }}`

	docTasks = `{{ if list := note.PartialRenderer().FirstList(); list }}
  <ul class="tasks">
    {{ range i, item := list.Items }}
      <li class="tasks__item tasks__item--{{ item.Task }}"
          data-mark="{{ item.TaskMark }}">
        {{ item.Text }}
      </li>
    {{ end }}
  </ul>
{{ end }}`

	docCsv = `{{ if b := note.PartialRenderer().CodeBlocks("csv"); len(b) > 0 }}
  {{ if rows := parseCSV(b[0].Content); rows }}
    <table>
      {{ range i, row := rows }}
        <tr>
          {{ range j, cell := row }}
            <td>{{ cell }}</td>
          {{ end }}
        </tr>
      {{ end }}
    </table>
  {{ end }}
{{ end }}`

	docIfExamples = `{{ if subtitle := note.M().GetString("subtitle", ""); subtitle != "" }}
  <p class="subtitle">{{ subtitle }}</p>
{{ else }}
  <p class="subtitle">{{ note.Title() }}</p>
{{ end }}

{{ blog := nvs.ByGlob("blog/*.md").Public() }}
{{ if latest := blog.SortBy("CreatedAt").Desc().First(); latest }}
  Latest post: <a href="{{ latest.Permalink() }}">{{ latest.Title() }}</a>
{{ end }}

{{ if faq := note.PartialRenderer().Section("FAQ"); faq }}
  {{ faq.ContentHTML }}
{{ end }}`

	docQuery = `{{ blog := nvs.ByGlob("blog/*.md").Public() }}
{{ posts := blog.SortBy("CreatedAt").Desc().Limit(5).All() }}
{{ range i, post := posts }}
  <a href="{{ post.Permalink() }}">{{ post.Title() }}</a>
{{ end }}`

	docSectionComponent = `{{ block @lid(path="") }}
  {{ if part := nvs.ByPath(path); part }}
    <section class="@did">
      <h2>{{ part.Title() }}</h2>
      {{ part.HTMLString() }}
    </section>
  {{ end }}
{{ end }}`

	docFaqComponent = `{{ block @lid(path="") }}
  {{ if faq := nvs.ByPath(path); faq }}
    {{ range i, s := faq.PartialRenderer().Sections(2) }}
      <details class="@did">
        <summary>{{ s.TitleHTML }}</summary>
        {{ s.ContentHTML }}
      </details>
    {{ end }}
  {{ end }}
{{ end }}`

	docSeveralPage = `{{ note.HTMLString() }}
{{ yield components_section(path="/blocks/pricing.md") }}
{{ yield components_faq(path="/blocks/faq.md") }}`

	docBase = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ block title() }}{{ note.Title() }}{{ end }}</title>
</head>
<body>
  <header>My site</header>
  <main>
    {{ block main() }}
      {{ note.HTMLString() }}
    {{ end }}
  </main>
  <footer>
    {{ block footer() }}© My site{{ end }}
  </footer>
</body>
</html>`

	docArticle = `{{ extends "base" }}

{{ block title() }}{{ note.Title() }} · Blog{{ end }}

{{ block main() }}
  <article>
    <h1>{{ note.Title() }}</h1>
    {{ note.HTMLString() }}
    {{ yield components_button(label="All posts", url="/blog") }}
  </article>
{{ end }}`

	docBaseImport = `{{ import "components/header" }}
<!DOCTYPE html>
<html lang="en">
<body>
  {{ yield components_header() }}
  <main>{{ block main() }}{{ end }}</main>
</body>
</html>`
)

func TestTemplatesDocSnippetsInDocs(t *testing.T) {
	var snippets []docSnippet
	for name, src := range map[string]string{
		"html injections":   docHTMLInjections,
		"seo":               docSeo,
		"toc":               docToc,
		"tasks":             docTasks,
		"csv":               docCsv,
		"if examples":       docIfExamples,
		"section component": docSectionComponent,
		"faq component":     docFaqComponent,
		"several notes":     docSeveralPage,
		"base":              docBase,
		"article":           docArticle,
		"base with import":  docBaseImport,
	} {
		snippets = append(snippets, docSnippet{name: name, src: src})
	}
	requireSnippetsInDocs(t, []string{"en/user/templates", "ru/user/templates"}, snippets)
	requireSnippetsInDocs(t, []string{"en/user/templates", "ru/user/templates-advanced"}, []docSnippet{{name: "query", src: docQuery}})

	var constructs []docSnippet
	for name, src := range map[string]string{
		"featured lib":    docFeaturedLib,
		"featured caller": docFeaturedCaller,
		"stats lib":       docStatsLib,
		"stats caller":    docStatsCaller,
		"chart widget":    docChartWidget,
		"related note":    docRelatedNote,
		"stats in try":    docStatsInTry,
	} {
		constructs = append(constructs, docSnippet{name: name, src: src})
	}
	requireSnippetsInDocs(t, []string{"en/user/templates", "ru/user/jet"}, constructs)
}

func loadDocNotes(t *testing.T, files map[string]string) *model.NoteViews {
	t.Helper()
	var sources []mdloader.SourceFile
	for path, content := range files {
		sources = append(sources, mdloader.SourceFile{Path: path, Content: []byte(content)})
	}
	notes, err := mdloader.Load(mdloader.Options{Sources: sources, Log: &logger.TestLogger{}})
	require.NoError(t, err)
	return notes
}

const docLayoutID = "/p"

func renderDocLayout(t *testing.T, sources []model.LayoutSourceFile, vars jet.VarMap) string {
	t.Helper()
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)
	require.Empty(t, layouts.Map[docLayoutID].Warnings)
	out, err := renderWith(t, layouts, docLayoutID, vars)
	require.NoError(t, err)
	return squash(out)
}

func docVars(notes *model.NoteViews, notePath string) jet.VarMap {
	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(notes.PathMap[notePath]))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(notes, "live"))
	vars["title"] = reflect.ValueOf("Guide | Site")
	vars["publicURL"] = reflect.ValueOf("https://example.com")
	vars["htmlInjectionsHead"] = reflect.ValueOf([]db.HtmlInjection{{Content: `<script src="/a.js"></script>`}})
	vars["htmlInjectionsBodyEnd"] = reflect.ValueOf([]db.HtmlInjection{{Content: `<script>track()</script>`}})
	return vars
}

const docGuideNote = `---
title: Guide
toc: true
---
Intro.

- [ ] open
- [x] done
- [/] started

## Install

Run it.

## FAQ

Ask us.

` + "```csv\nname,age\nAnn,30\n```\n"

func TestTemplatesDocExamples(t *testing.T) {
	notes := loadDocNotes(t, map[string]string{
		"guide.md":          docGuideNote,
		"post.md":           "---\ntitle: Q&A\n---\nHello.",
		"blog/old.md":       "---\ntitle: Old\ncreated_at: 2024-01-01\nfree: true\n---\nOld.",
		"blog/new.md":       "---\ntitle: New\ncreated_at: 2024-05-01\nfree: true\n---\nNew.",
		"blocks/pricing.md": "---\ntitle: Pricing\n---\nFrom $5 a month.",
		"blocks/faq.md":     "---\ntitle: FAQ\n---\n## Is it free?\n\nYes.\n\n## Can I leave?\n\nAny time.",
	})
	type docCase struct {
		name    string
		note    string
		sources []model.LayoutSourceFile
		want    string
	}
	tests := []docCase{
		{
			name:    "html injections",
			note:    "post.md",
			sources: []model.LayoutSourceFile{page("/p", docHTMLInjections)},
			want:    `<head> <title>Guide | Site</title> <script src="/a.js"></script> </head> <body> <p>Hello.</p> <script>track()</script> </body>`,
		},
		{
			name:    "seo",
			note:    "post.md",
			sources: []model.LayoutSourceFile{page("/p", docSeo)},
			want:    `<head> <link rel="canonical" href="https://example.com/post"> </head>`,
		},
		{
			name:    "toc",
			note:    "guide.md",
			sources: []model.LayoutSourceFile{page("/p", docToc)},
			want:    `<nav class="toc"> <a class="toc__item toc__item--1" href="#install"> Install </a> <a class="toc__item toc__item--1" href="#faq"> FAQ </a> </nav> <section id="install"> <h2>Install</h2> <p>Run it.</p> </section> <section id="faq"> <h2>FAQ</h2> <p>Ask us.</p> <pre><code class="language-csv">name,age Ann,30 </code></pre> </section>`,
		},
		{
			name:    "tasks",
			note:    "guide.md",
			sources: []model.LayoutSourceFile{page("/p", docTasks)},
			want:    `<ul class="tasks"> <li class="tasks__item tasks__item--todo" data-mark=" "> open </li> <li class="tasks__item tasks__item--done" data-mark="x"> done </li> <li class="tasks__item tasks__item--done" data-mark="/"> started </li> </ul>`,
		},
		{
			name:    "csv",
			note:    "guide.md",
			sources: []model.LayoutSourceFile{page("/p", docCsv)},
			want:    `<table> <tr> <td>name</td> <td>age</td> </tr> <tr> <td>Ann</td> <td>30</td> </tr> </table>`,
		},
		{
			name:    "if examples",
			note:    "guide.md",
			sources: []model.LayoutSourceFile{page("/p", docIfExamples)},
			want:    `<p class="subtitle">Guide</p> Latest post: <a href="/blog/new">New</a> <p>Ask us.</p> <pre><code class="language-csv">name,age Ann,30 </code></pre>`,
		},
		{
			name:    "query",
			note:    "post.md",
			sources: []model.LayoutSourceFile{page("/p", docQuery)},
			want:    `<a href="/blog/new">New</a> <a href="/blog/old">Old</a>`,
		},
		{
			name: "several notes",
			note: "post.md",
			sources: []model.LayoutSourceFile{
				page("/p", docSeveralPage),
				page("/components/section", docSectionComponent),
				page("/components/faq", docFaqComponent),
			},
			want: `<p>Hello.</p> <section class="components-section"> <h2>Pricing</h2> <p>From $5 a month.</p> </section> <details class="components-faq"> <summary>Is it free?</summary> <p>Yes.</p> </details> <details class="components-faq"> <summary>Can I leave?</summary> <p>Any time.</p> </details>`,
		},
		{
			name: "extends",
			note: "post.md",
			sources: []model.LayoutSourceFile{
				page("/base", docBase),
				page("/p", docArticle),
				page("/components/button", componentsDocButton),
			},
			want: `<!DOCTYPE html> <html lang="en"> <head> <meta charset="utf-8"> <title>Q&amp;A · Blog</title> </head> <body> <header>My site</header> <main> <article> <h1>Q&amp;A</h1> <p>Hello.</p> <a class="components-button" href="/blog">All posts</a> </article> </main> <footer> © My site </footer> </body> </html>`,
		},
		{
			name: "base with defaults",
			note: "post.md",
			sources: []model.LayoutSourceFile{
				page("/p", docBase),
				page("/article", docArticle),
				page("/components/button", componentsDocButton),
			},
			want: `<!DOCTYPE html> <html lang="en"> <head> <meta charset="utf-8"> <title>Q&amp;A</title> </head> <body> <header>My site</header> <main> <p>Hello.</p> </main> <footer> © My site </footer> </body> </html>`,
		},
		{
			name: "base with import",
			note: "post.md",
			sources: []model.LayoutSourceFile{
				page("/framed", docBaseImport),
				page("/p", `{{ extends "framed" }}{{ block main() }}M{{ end }}`),
				page("/components/header", `{{ block @lid() }}<header>H</header>{{ end }}`),
			},
			want: `<!DOCTYPE html> <html lang="en"> <body> <header>H</header> <main>M</main> </body> </html>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderDocLayout(t, tt.sources, docVars(notes, tt.note))
			require.Equal(t, tt.want, out)
		})
	}
}

func TestTemplatesDocBaseSlotForms(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{
			name: "bare yield",
			base: `<main>{{ yield main() }}</main>`,
			want: `<main>M</main>`,
		},
		{
			name: "block with default",
			base: `<main>{{ block main() }}default{{ end }}</main>`,
			want: `<main>M</main>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layouts := testLoadLayouts(t, []model.LayoutSourceFile{
				page("/base", tt.base),
				page("/page", `{{ extends "base" }}{{ block main() }}M{{ end }}`),
			})
			require.NotNil(t, layouts.Map["/base"].View, "%v", layouts.Map["/base"].Warnings)
			require.Equal(t, tt.want, renderLayout(t, layouts, "/page"))
		})
	}
}

// The app layout in docs/{en,ru}/user/spa.md.
const docSpaLayout = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
</head>
<body>
  <div id="app"></div>

  <script type="application/json" id="app-data">
    {{ json(map(
      "path", note.Path(),
      "content", note.ContentString(),
      "editable", currentUser.IsAdmin()
    )) }}
  </script>
  <script src="{{ asset("app.js") }}"></script>
</body>
</html>`

func TestSpaDocLayout(t *testing.T) {
	requireSnippetsInDocs(t, []string{"en/user/spa", "ru/user/spa"}, []docSnippet{{name: "app layout", src: docSpaLayout}})

	notes := loadDocNotes(t, map[string]string{
		"boards/todo.md": "---\ntitle: Todo\n---\n- [ ] ship </script><b>it</b> & more\n",
	})
	vars := docVars(notes, "boards/todo.md")
	vars["title"] = reflect.ValueOf("Todo")
	vars["currentUser"] = reflect.ValueOf(map[string]interface{}{
		"IsAdmin": func() bool { return true },
	})

	out := renderDocLayout(t, []model.LayoutSourceFile{page("/p", docSpaLayout)}, vars)
	want := `<!DOCTYPE html> <html lang="en"> <head> <meta charset="utf-8"> <title>Todo</title> </head> <body> ` +
		`<div id="app"></div> <script type="application/json" id="app-data"> ` +
		`{"content":"---\ntitle: Todo\n---\n- [ ] ship \u003c/script\u003e\u003cb\u003eit\u003c/b\u003e \u0026 more\n",` +
		`"editable":true,"path":"boards/todo.md"} </script> <script src="app.js"></script> </body> </html>`
	require.Equal(t, want, out)
}

// The app layout with the standard site chrome in docs/{en,ru}/user/spa.md.
const docSpaChrome = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main id="app"></main>
  <script type="application/json" id="app-data">
    {{ json(map(
      "path", note.Path(),
      "content", note.ContentString(),
      "editable", currentUser.IsAdmin()
    )) }}
  </script>
  <script src="{{ asset("app.js") }}"></script>
  {{ defaultTemplate.Footer() }}
</body>
</html>`

// The chrome skeleton in docs/{en,ru}/user/templates.md.
const docTemplatesChrome = `<head>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main id="app"></main>
  {{ defaultTemplate.Footer() }}
</body>`

func chromeDocVars(t *testing.T, admin bool, withChrome bool) jet.VarMap {
	t.Helper()
	notes := loadDocNotes(t, map[string]string{
		"boards/todo.md": "---\ntitle: Todo\n---\n- [ ] ship\n",
	})
	vars := docVars(notes, "boards/todo.md")
	vars["title"] = reflect.ValueOf("Todo")
	vars["currentUser"] = reflect.ValueOf(map[string]interface{}{
		"IsAdmin": func() bool { return admin },
	})
	header := model.SafeHTML("")
	footer := model.SafeHTML("")
	if withChrome {
		header = `<header><div mol_view_root="$trip2g_user_space"></div></header>`
		footer = `<footer>F</footer>`
	}
	defaultTemplate := map[string]interface{}{
		"Styles":           func() model.SafeHTML { return `<link rel="stylesheet" href="/s.css">` },
		"UserSpaceScripts": func() model.SafeHTML { return `<script src="/u.js" defer></script>` },
		"Header":           func() model.SafeHTML { return header },
		"Footer":           func() model.SafeHTML { return footer },
	}
	vars["defaultTemplate"] = reflect.ValueOf(defaultTemplate)
	return vars
}

func TestSpaDocChrome(t *testing.T) {
	requireSnippetsInDocs(t, []string{"en/user/spa", "ru/user/spa"}, []docSnippet{{name: "chrome layout", src: docSpaChrome}})
	requireSnippetsInDocs(t, []string{"en/user/templates", "ru/user/templates"}, []docSnippet{{name: "chrome", src: docTemplatesChrome}})

	head := `<!DOCTYPE html> <html lang="en"> <head> <meta charset="utf-8"> <title>Todo</title> ` +
		`<link rel="stylesheet" href="/s.css"> <script src="/u.js" defer></script> </head> <body> `
	data := `<script type="application/json" id="app-data"> ` +
		`{"content":"---\ntitle: Todo\n---\n- [ ] ship\n","editable":%s,"path":"boards/todo.md"} </script> ` +
		`<script src="app.js"></script> `
	tests := []struct {
		name       string
		admin      bool
		withChrome bool
		want       string
	}{
		{
			name:       "admin with header and footer",
			admin:      true,
			withChrome: true,
			want: head + `<header><div mol_view_root="$trip2g_user_space"></div></header> <main id="app"></main> ` +
				strings.Replace(data, "%s", "true", 1) + `<footer>F</footer> </body> </html>`,
		},
		{
			name:  "visitor on a page without header or footer notes",
			admin: false,
			want: head + `<main id="app"></main> ` +
				strings.Replace(data, "%s", "false", 1) + `</body> </html>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := chromeDocVars(t, tt.admin, tt.withChrome)
			out := renderDocLayout(t, []model.LayoutSourceFile{page("/p", docSpaChrome)}, vars)
			require.Equal(t, tt.want, out)
		})
	}

	t.Run("templates skeleton", func(t *testing.T) {
		vars := chromeDocVars(t, true, true)
		out := renderDocLayout(t, []model.LayoutSourceFile{page("/p", docTemplatesChrome)}, vars)
		want := `<head> <link rel="stylesheet" href="/s.css"> <script src="/u.js" defer></script> </head> ` +
			`<body> <header><div mol_view_root="$trip2g_user_space"></div></header> <main id="app"></main> ` +
			`<footer>F</footer> </body>`
		require.Equal(t, want, out)
	})
}
