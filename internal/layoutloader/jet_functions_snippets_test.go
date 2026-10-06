package layoutloader

import (
	"html/template"
	"os"
	"reflect"
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

type docSnippet struct {
	name   string
	setup  string
	src    string
	caller string
	want   string
}

// Every src below is copied verbatim into both language versions of
// docs/{en,ru}/user/jet-functions.md; requireSnippetsInDocs fails when a page drifts.
func jetFunctionsSnippets() []docSnippet {
	return []docSnippet{
		{
			name: "escaped by default",
			src: `{{ "<b>bold</b>" }}            {* → &lt;b&gt;bold&lt;/b&gt; *}
{{ "<b>bold</b>" | unsafe }}   {* → <b>bold</b> *}`,
			want: `&lt;b&gt;bold&lt;/b&gt; <b>bold</b>`,
		},
		{
			name: "three call forms",
			src: `{{ upper("hello") }}                {* → HELLO *}
{{ "hello" | upper }}               {* → HELLO *}
{{ "a-b-c" | replace: "-", " ", -1 }}
{* replace("a-b-c", "-", " ", -1) → a b c *}
{{ "Hello" | hasPrefix: "He" }}     {* → true *}
{{ "a-b" | replace: "-", " ", -1 | upper }}
{* pipes chain → A B *}`,
			want: `HELLO HELLO a b c true A B`,
		},
		{
			name: "strings",
			src: `{{ lower("ABC") }}                  {* → abc *}
{{ trimSpace("  x  ") }}            {* → x *}
{{ repeat("ab", 3) }}               {* → ababab *}
{{ replace("aaa", "a", "b", 2) }}   {* → bba *}

{{ range i, part := split("a,b,c", ",") }}
  [{{ part }}]
{{ end }}
{* → [a] [b] [c] *}

{{ if hasSuffix(note.Permalink(), "/faq") }}
  <a href="/faq/contact">Ask a question</a>
{{ end }}`,
			want: `abc x ababab bba [a] [b] [c] <a href="/faq/contact">Ask a question</a>`,
		},
		{
			name: "isset",
			src: `{{ m := map("a", 1) }}
{{ isset(m["a"]) }}        {* → true *}
{{ isset(m["z"]) }}        {* → false: missing map key *}
{{ isset(undefinedVar) }}  {* → false, no error *}`,
			want: `true false false`,
		},
		{
			name: "map and ints",
			src: `{{ links := map("docs", "/docs", "blog", "/blog") }}
{{ links["blog"] }}  {{ links.blog }}   {* both → /blog *}

{{ range i, n := ints(1, 4) }}
  {{ n }}
{{ end }}
{* → 1 2 3 *}`,
			want: `/blog /blog 1 2 3`,
		},
		{
			name: "index and slice with variables",
			src: `{{ colors := slice("red", "green", "blue", "black") }}
{{ colors[1] }}             {* → green *}
{{ colors[1:3] }}           {* → [green blue] *}

{{ from := 1 }}
{{ to := len(colors) - 1 }}
{{ colors[from:to] }}       {* → [green blue] *}
{{ colors[from:] }}         {* → [green blue black] *}
{{ colors[:to] }}           {* → [red green blue] *}`,
			want: `green [green blue] [green blue] [green blue black] [red green blue]`,
		},
		{
			name: "string slice",
			src:  `{{ word := "hello" }}{{ word[1:4] }}`,
			want: `ell`,
		},
		{
			name:  "minus with spaces in an index",
			setup: `{{ colors := slice("red", "green", "blue", "black") }}`,
			src:   `colors[len(colors) - 2:]`,
			want:  `[blue black]`,
		},
		{
			name: "json over several lines",
			src: `<script>
  window.pageData = {{ json(map(
    "title", note.Title(),
    "tags", note.Tags()
  )) }};
</script>`,
			want: `<script> window.pageData = {"tags":["go","jet"],"title":"Q\u0026A"}; </script>`,
		},
		{
			name: "writeJson",
			src: `<script type="application/json" id="data">
  {{ writeJson(note.M().Raw()) }}
</script>`,
			want: `<script type="application/json" id="data"> {"title":"Q\u0026A"} </script>`,
		},
		{
			name: "includeIfExists",
			src: `{{ includeIfExists("partials/banner") }}

{{ if !includeIfExists("partials/sidebar", note) }}
  <p>No sidebar</p>
{{ end }}`,
			want: `<div class="banner">Sale</div> <p>No sidebar</p>`,
		},
		{
			name: "escaping filters",
			src: `{{ s := "<b>Tom & Jerry</b>" }}
{{ s }}               {* → &lt;b&gt;Tom &amp; Jerry&lt;/b&gt; *}
{{ s | html }}        {* the same *}
{{ s | safeHtml }}    {* the same *}
{{ s | unsafe }}      {* → <b>Tom & Jerry</b> *}
{{ s | raw }}         {* the same as unsafe *}`,
			want: `&lt;b&gt;Tom &amp; Jerry&lt;/b&gt; &lt;b&gt;Tom &amp; Jerry&lt;/b&gt; ` +
				`&lt;b&gt;Tom &amp; Jerry&lt;/b&gt; <b>Tom & Jerry</b> <b>Tom & Jerry</b>`,
		},
		{
			name:  "html is a value",
			setup: `{{ s := "<b>Tom & Jerry</b>" }}`,
			src: `{{ e := html(s) }}
<p title="{{ e }}">{{ e }}</p>
{* both escaped once: &lt;b&gt;Tom &amp; Jerry&lt;/b&gt; *}

<pre>{{ html(note.HTMLString()) }}</pre>
{* the note's HTML as visible tags *}`,
			want: `<p title="&lt;b&gt;Tom &amp; Jerry&lt;/b&gt;">&lt;b&gt;Tom &amp; Jerry&lt;/b&gt;</p> ` +
				`<pre>&lt;p&gt;Hi&lt;/p&gt;</pre>`,
		},
		{
			name: "url",
			src: `{{ term := "a b&c/d" }}
<a href="/search?q={{ term | url }}">Search</a>
{* → href="/search?q=a+b%26c%2Fd" *}`,
			want: `<a href="/search?q=a+b%26c%2Fd">Search</a>`,
		},
		{
			name: "safeJs",
			src: `<script>
  var title = '{{ note.Title() | safeJs }}';
</script>`,
			want: `<script> var title = 'Q\u0026A'; </script>`,
		},
		{
			name: "whitespace trim",
			src: `{{ note.Title() }}                  {* print an expression *}
{* a comment, never rendered *}
<li>   {{- note.Title() -}}   </li>
{* {{- and -}} trim whitespace on that side *}`,
			want: `Q&amp;A <li>Q&amp;A</li>`,
		},
		{
			name:  "if",
			setup: `{{ n := 2 }}{{ links := map("blog", "/blog") }}`,
			src: `{{ if n == 1 }}
  one
{{ else if n == 2 }}
  two
{{ else }}
  many
{{ end }}

{{ if v, ok := links["blog"]; ok }}
  <a href="{{ v }}">Blog</a>
{{ end }}`,
			want: `two <a href="/blog">Blog</a>`,
		},
		{
			name:  "range",
			setup: `{{ links := map("blog", "/blog") }}{{ posts := slice() }}`,
			src: `{{ range i, tag := note.Tags() }}
  <span>{{ tag }}</span>
{{ end }}

{{ range key, value := links }}
  {{ key }} → {{ value }}
{{ end }}

{{ range i, post := posts }}
  <a href="{{ post.Permalink() }}">{{ post.Title() }}</a>
{{ else }}
  <p>Nothing here.</p>
{{ end }}`,
			want: `<span>go</span> <span>jet</span> blog → /blog <p>Nothing here.</p>`,
		},
		{
			name: "block wrapping content",
			src: `{{ block panel(title="") }}
  <section>
    <h2>{{ title }}</h2>
    {{ yield content }}
  </section>
{{ end }}

{{ yield panel(title="Related") content }}
  <p>Anything here becomes the panel body.</p>
{{ end }}`,
			want: `<section> <h2></h2> </section> ` +
				`<section> <h2>Related</h2> <p>Anything here becomes the panel body.</p> </section>`,
		},
		{
			name: "try and catch",
			src: `{{ try }}
  {{ stats := exec("lib/stats", "blog") }}
  <p>{{ stats.count }} posts</p>
{{ catch err }}
  {{ if currentUser.IsAdmin() }}
    <p>{{ err.Error() }}</p>
  {{ end }}
{{ end }}`,
			want: `<p>1 posts</p>`,
		},
		{
			name:  "coalesce over several lines",
			setup: `{{ videos := map("en", "intro-en.mp4") }}`,
			src: `{{ coalesce(
  note.M().GetString("subtitle", ""),
  note.Description(),
  note.Title()
) }}`,
			want: `Q&amp;A`,
		},
		{
			name: "default template frame",
			src: `<head>
  <title>{{ title }}</title>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main>{{ note.HTMLString() }}</main>
  {{ defaultTemplate.Footer() }}
  {{ if currentUser.IsAdmin() }}
    <a href="/admin">Admin</a>
  {{ end }}
</body>`,
			want: `<head> <title>Q&amp;A | Site</title> <link rel="stylesheet" href="/s.css"> ` +
				`</head> <body> <header>H</header> <main><p>Hi</p></main> <footer>F</footer> ` +
				`<a href="/admin">Admin</a> </body>`,
		},
		{
			name: "import and include",
			src: `{{ import "blocks" }}
{* load the blocks of _layouts/blocks.html, render nothing *}

{{ include "partials/footer" }}
{* render another template here *}

{{ include "partials/card" note }}
{* … with ` + "`note`" + ` as ` + "`.`" + ` inside it *}`,
			want: `<footer>F</footer> <div>Q&amp;A</div>`,
		},
		{
			name: "extends",
			src: `{{ extends "base" }}

{{ block main() }}
  <p>This replaces the main block of base.html</p>
{{ end }}`,
			want: `<!DOCTYPE html> <html lang="en"> <head> <meta charset="utf-8"> <title>Q&amp;A</title> </head> ` +
				`<body> <header>My site</header> <main> <p>This replaces the main block of base.html</p> </main> ` +
				`<footer> © My site </footer> </body> </html>`,
		},
		{
			name: "return",
			src: `{* lib/featured.html: the value exec gets *}
{{ return nvs.ByGlob(. + "/*.md").Public().Limit(3).All() }}`,
			caller: `{{ range _, p := exec("snippet", "blog") }}{{ p.Title() }}{{ end }}`,
			want:   `A`,
		},
		{
			name: "language switcher",
			src: `{{ range i, alt := note.LangAlternativesList() }}
  <a href="{{ alt.PermalinkEncoded() }}" hreflang="{{ alt.Lang() }}">
    {{ alt.LangName() }}
  </a>
{{ end }}`,
			want: ``,
		},
		{
			name: "lookup in if",
			src: `{{ if about := nvs.ByPermalink("/about"); about }}
  <a href="{{ about.PermalinkEncoded() }}">{{ about.Title() }}</a>
{{ end }}`,
			want: `<a href="/about">About &amp; us</a>`,
		},
	}
}

func readDoc(t *testing.T, page string) string {
	t.Helper()
	data, err := os.ReadFile("../../docs/" + page + ".md")
	require.NoError(t, err)
	return string(data)
}

func requireSnippetsInDocs(t *testing.T, pages []string, snippets []docSnippet) {
	t.Helper()
	for _, page := range pages {
		doc := readDoc(t, page)
		for _, s := range snippets {
			require.Contains(t, doc, s.src, "docs/%s.md lost the %q example", page, s.name)
		}
	}
}

func jetFunctionsPages() []string {
	return []string{"en/user/jet-functions", "ru/user/jet-functions"}
}

func snippetVars(admin bool) jet.VarMap {
	notes := model.NewNoteViews()
	notes.PathMap["about.md"] = &model.NoteView{Path: "about.md", Title: "About & us", Permalink: "/about", Free: true}
	notes.PathMap["blog/a.md"] = &model.NoteView{Path: "blog/a.md", Title: "A", Permalink: "/blog/a", Free: true}
	notes.Map["/about"] = notes.PathMap["about.md"]

	page := &model.NoteView{
		Path:      "help/faq.md",
		Title:     "Q&A",
		Permalink: "/help/faq",
		HTML:      template.HTML("<p>Hi</p>"),
		Tags:      []string{"go", "jet"},
		RawMeta:   map[string]interface{}{"title": "Q&A"},
	}

	defaultTemplate := map[string]interface{}{
		"Styles":           func() model.SafeHTML { return `<link rel="stylesheet" href="/s.css">` },
		"UserSpaceScripts": func() model.SafeHTML { return "" },
		"Header":           func() model.SafeHTML { return "<header>H</header>" },
		"Footer":           func() model.SafeHTML { return "<footer>F</footer>" },
	}

	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(page))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(notes, "live"))
	vars["title"] = reflect.ValueOf("Q&A | Site")
	vars["currentUser"] = reflect.ValueOf(map[string]interface{}{
		"IsAdmin": func() bool { return admin },
	})
	vars["defaultTemplate"] = reflect.ValueOf(defaultTemplate)
	return vars
}

func renderSnippets(t *testing.T, snippets []docSnippet, extra []model.LayoutSourceFile) {
	t.Helper()
	for _, s := range snippets {
		t.Run(s.name, func(t *testing.T) {
			src := s.setup + s.src
			if strings.HasPrefix(s.src, "colors[") {
				src = s.setup + "{{ " + s.src + " }}"
			}
			sources := append([]model.LayoutSourceFile{page("/snippet", src)}, extra...)
			entry := "/snippet"
			if s.caller != "" {
				sources = append(sources, page("/caller", s.caller))
				entry = "/caller"
			}
			layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
			require.NoError(t, err)

			out, err := renderWith(t, layouts, entry, snippetVars(true))
			require.NoError(t, err)
			require.Equal(t, s.want, squash(out))
		})
	}
}

func TestJetFunctionsDocSnippets(t *testing.T) {
	requireSnippetsInDocs(t, jetFunctionsPages(), jetFunctionsSnippets())

	extra := []model.LayoutSourceFile{
		page("/partials/banner", `<div class="banner">Sale</div>`),
		page("/lib/stats", `{{ return map("count", len(nvs.ByGlob(. + "/*.md").All())) }}`),
		page("/blocks", `{{ block unused() }}{{ end }}`),
		page("/partials/footer", `<footer>F</footer>`),
		page("/partials/card", `<div>{{ .Title() }}</div>`),
		page("/base", docBase),
	}
	renderSnippets(t, jetFunctionsSnippets(), extra)
}

func TestJetSyntaxDocSnippets(t *testing.T) {
	snippets := []docSnippet{{
		name:  "slice with variables",
		setup: `{{ items := slice("a", "b", "c", "d") }}`,
		src: `{{ from := 1 }}
{{ to := len(items) - 1 }}
{{ range i, item := items[from:to] }}
  {{ item }}
{{ end }}`,
		want: `b c`,
	}}
	requireSnippetsInDocs(t, []string{"ru/user/jet"}, snippets)
	renderSnippets(t, snippets, nil)
}
