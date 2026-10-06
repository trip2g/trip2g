package layoutloader

import (
	"bytes"
	"encoding/json"
	"html/template"
	"reflect"
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

const presignedURL = "https://s3.example.com/bucket/bg.png?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=abc%2F20261006&X-Amz-Signature=deadbeef"

func renderEscapeLayout(t *testing.T, content string, assets map[string]*model.NoteAssetReplace, vars jet.VarMap) string {
	t.Helper()

	sources := []model.LayoutSourceFile{{
		ID:      "/page",
		Path:    "_layouts/page.html",
		Content: content,
		Assets:  assets,
	}}

	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)
	require.NotNil(t, layouts.Map["/page"].View, "layout must parse: %v", layouts.Map["/page"].Warnings)

	var buf bytes.Buffer
	err = layouts.Map["/page"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)
	return buf.String()
}

func escapeTestNote() *templateviews.Note {
	return templateviews.NewNote(&model.NoteView{
		Title:   `Tom & "Jerry" <3`,
		Content: []byte("# Hi\n</textarea><script>alert(1)</script>"),
		HTML:    template.HTML(`<p>Hello <strong>world</strong></p>`),
	})
}

func TestEscapeByDefault(t *testing.T) {
	vars := jet.VarMap{}
	vars.Set("note", escapeTestNote())
	vars.Set("x", `<b>"a" & 'b'</b>`)
	vars.Set("section", model.NoteViewSection{
		TitleHTML:   "<em>Title</em>",
		ContentHTML: "<p>Body</p>",
	})
	vars.Set("code", model.NoteViewCodeBlock{HTML: `<pre><code>x</code></pre>`})

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"markdown in a textarea cannot close it",
			`<textarea>{{ note.ContentString() }}</textarea>`,
			`<textarea># Hi
&lt;/textarea&gt;&lt;script&gt;alert(1)&lt;/script&gt;</textarea>`,
		},
		{"text is escaped", `{{ note.Title() }}`, `Tom &amp; &#34;Jerry&#34; &lt;3`},
		{"rendered note HTML is not escaped", `{{ note.HTMLString() }}`, `<p>Hello <strong>world</strong></p>`},
		{"unsafe still writes raw", `{{ x | unsafe }}`, `<b>"a" & 'b'</b>`},
		{"raw still writes raw", `{{ x | raw }}`, `<b>"a" & 'b'</b>`},
		{"unsafe on safe HTML is harmless", `{{ note.HTMLString() | unsafe }}`, `<p>Hello <strong>world</strong></p>`},
		{"html pipe escapes once", `{{ x | html }}`, `&lt;b&gt;&#34;a&#34; &amp; &#39;b&#39;&lt;/b&gt;`},
		{"html call escapes once", `{{ html(x) }}`, `&lt;b&gt;&#34;a&#34; &amp; &#39;b&#39;&lt;/b&gt;`},
		{"html of note HTML escapes it once", `{{ html(note.HTMLString()) }}`, `&lt;p&gt;Hello &lt;strong&gt;world&lt;/strong&gt;&lt;/p&gt;`},
		{"url escapes once", `<a href="/s?q={{ x | url }}">`, `<a href="/s?q=%3Cb%3E%22a%22+%26+%27b%27%3C%2Fb%3E">`},
		{"safeHtml still escapes", `{{ x | safeHtml }}`, `&lt;b&gt;&#34;a&#34; &amp; &#39;b&#39;&lt;/b&gt;`},
		{"safeJs still escapes", `{{ x | safeJs }}`, `\u003Cb\u003E\"a\" \u0026 \'b\'\u003C/b\u003E`},
		{"writeJson still writes raw JSON", `{{ writeJson(x) }}`, "\"\\u003cb\\u003e\\\"a\\\" \\u0026 'b'\\u003c/b\\u003e\"\n"},
		{"section HTML is not escaped", `{{ section.TitleHTML }}{{ section.ContentHTML }}`, `<em>Title</em><p>Body</p>`},
		{"code block HTML is not escaped", `{{ code.HTML }}`, `<pre><code>x</code></pre>`},
		{"len of safe HTML", `{{ len(note.HTMLString()) }}`, "35"},
		{"if on safe HTML", `{{ if note.HTMLString() }}yes{{ end }}`, "yes"},
		{"safe HTML compares with a string", `{{ if section.TitleHTML == "<em>Title</em>" }}eq{{ end }}`, "eq"},
		{"string function on safe HTML returns text", `{{ upper(section.TitleHTML) }}`, `&lt;EM&gt;TITLE&lt;/EM&gt;`},
		{"concatenation is text", `{{ section.TitleHTML + "!" }}`, `&lt;em&gt;Title&lt;/em&gt;!`},
		{
			"safe HTML stays safe through a block parameter",
			`{{ block card(body="") }}<div>{{ body }}</div>{{ end }}{{ yield card(body=note.HTMLString()) }}{{ yield card(body="<br>") }}`,
			`<div></div><div><p>Hello <strong>world</strong></p></div><div>&lt;br&gt;</div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, renderEscapeLayout(t, tt.content, nil, vars))
		})
	}
}

func TestEscapeJSONInScript(t *testing.T) {
	vars := jet.VarMap{}
	vars.Set("data", map[string]any{"title": `</script><script>alert("x")</script>`, "n": 1})

	out := renderEscapeLayout(t, `<script>var d = {{ data | json }};</script>`, nil, vars)

	require.Equal(t, 1, strings.Count(out, "</script>"), "the value must not close the script element")
	payload := strings.TrimSuffix(strings.TrimPrefix(out, "<script>var d = "), ";</script>")

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload), &got))
	require.Equal(t, `</script><script>alert("x")</script>`, got["title"])
}

func TestEscapeAssetURL(t *testing.T) {
	assets := map[string]*model.NoteAssetReplace{
		"_layouts/bg.png":    {URL: presignedURL},
		"_layouts/quote.png": {URL: `https://s3.example.com/a"b c'd<e>.png`},
	}

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"presigned URL inside a style element is unchanged",
			`<style>.hero { background: url("{{ asset("bg.png") }}"); }</style>`,
			`<style>.hero { background: url("` + presignedURL + `"); }</style>`,
		},
		{
			"presigned URL inside an attribute is unchanged",
			`<img src="{{ asset("bg.png") }}">`,
			`<img src="` + presignedURL + `">`,
		},
		{
			"quotes, angle brackets and spaces are percent-encoded",
			`<img src="{{ asset("quote.png") }}">`,
			`<img src="https://s3.example.com/a%22b%20c%27d%3Ce%3E.png">`,
		},
		{
			"an unknown asset keeps its name",
			`{{ asset("missing.css") }}`,
			`missing.css`,
		},
		{
			"asset survives a map and coalesce",
			`{{ m := map("en", asset("bg.png")) }}{{ coalesce(m["ru"], m["en"]) }}`,
			presignedURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, renderEscapeLayout(t, tt.content, assets, nil))
		})
	}
}

func TestSafeHTMLIsAJetRenderer(t *testing.T) {
	require.True(t, reflect.TypeOf(model.SafeHTML("")).Implements(reflect.TypeOf((*jet.Renderer)(nil)).Elem()))
}
