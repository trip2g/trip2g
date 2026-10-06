package layoutloader

import (
	"bytes"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

func renderPage(t *testing.T, content string, vars jet.VarMap) string {
	t.Helper()

	sources := []model.LayoutSourceFile{{
		ID:      "/page",
		Path:    "_layouts/page.html",
		Content: content,
	}}

	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)

	var buf bytes.Buffer
	err = layouts.Map["/page"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	return buf.String()
}

func TestParseFuncs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"json object", `{{ d := parseJSON("{\"a\": [1, 2]}") }}{{ d["a"][1] }}`, "2"},
		{"json pipe", `{{ "\"x&y\"" | parseJSON | html }}`, "x&amp;y"},
		{"json nested fields", `{{ d := parseJSON("{\"a\": {\"b\": 1}}") }}{{ d.a.b }}`, "1"},
		{"json error is nil", `{{ if d := parseJSON("{oops"); d }}yes{{ else }}no{{ end }}`, "no"},
		{"json error compares to nil", `{{ parseJSON("{oops") == nil }}`, "true"},
		{"json null is nil", `{{ parseJSON("null") == nil }}`, "true"},
		{"yaml", `{{ d := parseYAML("title: Hi\nitems:\n  - one\n  - two\n") }}{{ d["title"] }} {{ d["items"][1] }}`, "Hi two"},
		{"yaml error is nil", `{{ if d := parseYAML("a: [1"); d }}yes{{ else }}no{{ end }}`, "no"},
		{"csv", `{{ range i, row := parseCSV("name,qty\napple,3\npear\n") }}{{ row[0] }}={{ len(row) }};{{ end }}`, "name=2;apple=2;pear=1;"},
		{"csv error is nil", `{{ parseCSV("a,\"b\nc") == nil }}`, "true"},
		{"non-string argument is nil", `{{ parseJSON(1) == nil }}`, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, renderPage(t, tt.content, nil))
		})
	}
}

func loadNote(t *testing.T, content string) *templateviews.Note {
	t.Helper()

	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{Path: "note.md", Content: []byte(content)}},
		Log:     &logger.TestLogger{},
	})
	require.NoError(t, err)

	return templateviews.NewNote(pages.PathMap["note.md"])
}

func TestCodeBlockChartTemplate(t *testing.T) {
	note := loadNote(t, "# Sales\n\n```mychart\n{\"labels\": [\"Q1\", \"Q2\"], \"values\": [3, 5]}\n```\n\n```mychart\nbroken\n```\n")

	content := `{{ range n, b := note.PartialRenderer().CodeBlocks("mychart") }}` +
		`{{ if d := parseJSON(b.Content); d }}<ul class="chart">` +
		`{{ range i, label := d.labels }}<li style="--v: {{ d.values[i] }}">{{ label | html }}</li>{{ end }}` +
		`</ul>{{ else }}<pre>{{ b.Content | html }}</pre>{{ end }}{{ end }}`

	vars := make(jet.VarMap)
	vars.Set("note", note)

	require.Equal(t,
		`<ul class="chart"><li style="--v: 3">Q1</li><li style="--v: 5">Q2</li></ul><pre>broken
</pre>`,
		renderPage(t, content, vars))
}

func TestTOCLinksMatchSectionIDs(t *testing.T) {
	note := loadNote(t, "---\ntoc: show\n---\n## Intro {#start}\n\na\n\n## Цены и тарифы\n\nb\n\n## Intro\n\nc\n")

	content := `{{ pr := note.PartialRenderer() }}` +
		`{{ range n, h := note.TOC() }}<a href="#{{ h.ID }}">{{ h.Text | html }}</a>` +
		`{{ s := pr.Section("#" + h.ID) }}[{{ s.ID }}]{{ end }}|` +
		`{{ pr.Section("intro").ID }}|` +
		`{{ range n, s := pr.Sections(2) }}{{ s.ID }},{{ end }}`

	vars := make(jet.VarMap)
	vars.Set("note", note)

	require.Equal(
		t,
		`<a href="#start">Intro</a>[start]<a href="#cenyi_i_tarifyi">Цены и тарифы</a>[cenyi_i_tarifyi]<a href="#intro">Intro</a>[intro]|start|start,cenyi_i_tarifyi,intro,`,
		renderPage(t, content, vars),
	)
}

func TestListTasksTemplate(t *testing.T) {
	note := loadNote(t, "- [ ] write\n- [x] test\n- [/] ship\n- note\n")

	content := `{{ l := note.PartialRenderer().FirstList() }}` +
		`{{ range n, item := l.Items }}{{ if item.Task }}{{ item.Task }}({{ item.TaskMark }}){{ else }}-{{ end }} {{ item.Text | html }};{{ end }}`

	vars := make(jet.VarMap)
	vars.Set("note", note)

	require.Equal(t, "todo( ) write;done(x) test;done(/) ship;- note;", renderPage(t, content, vars))
}

func TestImagesGalleryTemplate(t *testing.T) {
	note := loadNote(t, "![one](https://cdn.example.com/1.png \"First\")\n\n![[2.png|Two]]\n")

	content := `{{ range n, img := note.PartialRenderer().Images() }}` +
		`<img src="{{ img.URL | html }}" alt="{{ img.Alt | html }}">{{ if img.Title }}<i>{{ img.Title | html }}</i>{{ end }}{{ end }}`

	vars := make(jet.VarMap)
	vars.Set("note", note)

	require.Equal(t,
		`<img src="https://cdn.example.com/1.png" alt="one"><i>First</i><img src="2.png" alt="Two">`,
		renderPage(t, content, vars))
}

func TestCSVCodeBlockTemplate(t *testing.T) {
	note := loadNote(t, "```csv\nname,qty\napple,3\n```\n")

	content := `{{ if b := note.PartialRenderer().CodeBlocks("csv"); len(b) > 0 }}` +
		`{{ if rows := parseCSV(b[0].Content); rows }}` +
		`{{ range i, row := rows }}<tr>{{ range j, cell := row }}<td>{{ cell | html }}</td>{{ end }}</tr>{{ end }}` +
		`{{ end }}{{ end }}`

	vars := make(jet.VarMap)
	vars.Set("note", note)

	require.Equal(t, "<tr><td>name</td><td>qty</td></tr><tr><td>apple</td><td>3</td></tr>", renderPage(t, content, vars))
}
