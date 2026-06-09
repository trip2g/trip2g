package mdloader

import (
	"encoding/json"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type stubChartData struct{ rows json.RawMessage }

func (s stubChartData) ChartRows(_ int64, _ model.NoteViewChart) json.RawMessage { return s.rows }

func renderNote(t *testing.T, content string) string {
	t.Helper()
	log := logger.TestLogger{}
	pages, err := Load(Options{
		Sources: []SourceFile{{Path: "note.md", Content: []byte(content)}},
		Log:     &log,
	})
	require.NoError(t, err)
	p := pages.Map["/note"]
	require.NotNil(t, p)
	return string(p.HTML)
}

func TestChartRenderer_InlineSource(t *testing.T) {
	content := "```datachart\n" +
		`{"data":{"source":"inline","rows":[{"m":"Jan","v":1}]},"config":{"series":[{"type":"bar"}]}}` +
		"\n```\n"
	html := renderNote(t, content)

	require.Contains(t, html, `<div class="chart"`)
	require.Contains(t, html, `class="chart__data"`)
	require.Contains(t, html, `"data":[{"m":"Jan","v":1}]`) // rows baked in
	require.Contains(t, html, `"config":{"series":[{"type":"bar"}]}`)
	require.NotContains(t, html, "<pre><code") // not rendered as a code block
}

func TestChartRenderer_DefaultCodeBlockUnchanged(t *testing.T) {
	html := renderNote(t, "```go\nfmt.Println(1)\n```\n")
	require.Contains(t, html, `<pre><code class="language-go">`)
	require.Contains(t, html, "fmt.Println(1)")
	require.Contains(t, html, "</code></pre>")
}

func TestChartRenderer_URLSourceNoDataYet(t *testing.T) {
	content := "```datachart\n" +
		`{"data":{"source":"url","url":"http://x"},"config":{"series":[]}}` +
		"\n```\n"
	html := renderNote(t, content)
	require.Contains(t, html, `<div class="chart"`)
	require.NotContains(t, html, `data-src`) // url source has no client src
	require.NotContains(t, html, `"data":`)  // no provider → data omitted → loader
}

func TestChartRenderer_URLSourceUsesProvider(t *testing.T) {
	content := "```datachart\n" +
		`{"data":{"source":"url","url":"http://x"},"config":{"series":[{"type":"line"}]}}` +
		"\n```\n"
	log := logger.TestLogger{}
	pages, err := Load(Options{
		Sources:   []SourceFile{{Path: "note.md", Content: []byte(content)}},
		Log:       &log,
		ChartData: stubChartData{rows: json.RawMessage(`[{"a":1}]`)},
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.Contains(t, html, `"data":[{"a":1}]`, "provider rows should be baked into the page")
}

func TestStripWikilink(t *testing.T) {
	require.Equal(t, "x.csv", stripWikilink("[[x.csv]]"))
	require.Equal(t, "x.csv", stripWikilink("[[x.csv|Alias]]"))
	require.Equal(t, "x", stripWikilink("  [[ x ]] "))
	require.Equal(t, "plain.json", stripWikilink("plain.json"))
}

func TestWikilinkTargetFromMeta(t *testing.T) {
	require.Equal(t, "x.csv", wikilinkTargetFromMeta("[[x.csv]]"))
	require.Equal(t, "x.csv", wikilinkTargetFromMeta([]any{"[[x.csv]]", "[[y.csv]]"}))
	require.Empty(t, wikilinkTargetFromMeta(nil))
	require.Empty(t, wikilinkTargetFromMeta(42))
}

func TestFormatFromExt(t *testing.T) {
	require.Equal(t, "csv", formatFromExt("a/b/x.csv"))
	require.Equal(t, "json", formatFromExt("X.JSON"))
	require.Empty(t, formatFromExt("x.txt"))
}

func TestFindAssets_MarksFrontmatterDatachartAsset(t *testing.T) {
	content := "---\nchart_sales: \"[[sales.csv]]\"\n---\n\n" +
		"```datachart\n" +
		`{"data":{"source":"frontmatter","ref":"chart_sales"},"config":{"series":[{"type":"bar"}]}}` +
		"\n```\n"
	log := logger.TestLogger{}
	pages, err := Load(Options{
		Sources: []SourceFile{{Path: "note.md", Content: []byte(content)}},
		Log:     &log,
	})
	require.NoError(t, err)
	p := pages.Map["/note"]
	require.NotNil(t, p)

	_, ok := p.Assets["sales.csv"]
	require.Truef(t, ok, "frontmatter datachart asset should be marked; Assets=%v", p.Assets)
}

func TestFindAssets_InlineSourceMarksNothing(t *testing.T) {
	content := "```datachart\n" +
		`{"data":{"source":"inline","rows":[{"a":1}]},"config":{"series":[]}}` +
		"\n```\n"
	log := logger.TestLogger{}
	pages, err := Load(Options{
		Sources: []SourceFile{{Path: "note.md", Content: []byte(content)}},
		Log:     &log,
	})
	require.NoError(t, err)
	require.Empty(t, pages.Map["/note"].Assets)
}
