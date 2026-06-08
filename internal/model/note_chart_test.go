package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func parseToNoteView(t *testing.T, content string) *NoteView {
	t.Helper()
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader([]byte(content)))
	return &NoteView{Content: []byte(content), ast: doc}
}

func TestExtractCharts_URLSource(t *testing.T) {
	content := "```datachart\n" +
		`{"title":"Revenue","data":{"source":"url","url":"http://adapter/q","body":"{\"sql\":\"SELECT 1\"}"},"config":{"series":[{"type":"line"}]}}` +
		"\n```\n"
	n := parseToNoteView(t, content)
	n.extractCharts()

	require.Len(t, n.Charts, 1)
	c := n.Charts[0]
	require.Equal(t, 0, c.Index)
	require.Equal(t, "Revenue", c.Title)
	require.Equal(t, "url", c.Data.Source)
	require.Equal(t, "http://adapter/q", c.Data.URL)
	require.JSONEq(t, `{"sql":"SELECT 1"}`, c.Data.Body)
	require.JSONEq(t, `{"series":[{"type":"line"}]}`, string(c.Config))
	require.Len(t, c.Hash, 64) // url source is backend-cached → has a hash
}

func TestExtractCharts_FrontmatterSource(t *testing.T) {
	content := "```datachart\n" +
		`{"data":{"source":"frontmatter","ref":"chart_sales"},"config":{"series":[{"type":"bar"}]}}` +
		"\n```\n"
	n := parseToNoteView(t, content)
	n.extractCharts()

	require.Len(t, n.Charts, 1)
	c := n.Charts[0]
	require.Equal(t, "frontmatter", c.Data.Source)
	require.Equal(t, "chart_sales", c.Data.Ref)
	require.Empty(t, c.Hash, "frontmatter source is client-fetched → no backend cache key")
}

func TestExtractCharts_InternalSource(t *testing.T) {
	content := "```datachart\n" +
		`{"data":{"source":"internal","sql":"SELECT a, count(*) AS n FROM t GROUP BY a"},"config":{"series":[{"type":"pie"}]}}` +
		"\n```\n"
	n := parseToNoteView(t, content)
	n.extractCharts()

	require.Len(t, n.Charts, 1)
	c := n.Charts[0]
	require.Equal(t, "internal", c.Data.Source)
	require.Equal(t, "SELECT a, count(*) AS n FROM t GROUP BY a", c.Data.SQL)
	require.Len(t, c.Hash, 64)
}

func TestExtractCharts_InlineSource(t *testing.T) {
	content := "```datachart\n" +
		`{"data":{"source":"inline","rows":[{"m":"Jan","v":1}]},"config":{"series":[{"type":"bar"}]}}` +
		"\n```\n"
	n := parseToNoteView(t, content)
	n.extractCharts()

	require.Len(t, n.Charts, 1)
	c := n.Charts[0]
	require.Equal(t, "inline", c.Data.Source)
	require.JSONEq(t, `[{"m":"Jan","v":1}]`, string(c.Data.Rows))
	require.Empty(t, c.Hash, "inline data is bundled → no backend cache key")
}

func TestExtractCharts_MultipleBlocksIndexed(t *testing.T) {
	content := "```datachart\n" + `{"data":{"source":"url","url":"http://a"}}` + "\n```\n\n" +
		"text\n\n" +
		"```datachart\n" + `{"data":{"source":"url","url":"http://b"}}` + "\n```\n"
	n := parseToNoteView(t, content)
	n.extractCharts()

	require.Len(t, n.Charts, 2)
	require.Equal(t, 0, n.Charts[0].Index)
	require.Equal(t, "http://a", n.Charts[0].Data.URL)
	require.Equal(t, 1, n.Charts[1].Index)
	require.NotEqual(t, n.Charts[0].Hash, n.Charts[1].Hash)
}

func TestExtractCharts_HashStableForSameURLQuery(t *testing.T) {
	mk := func() *NoteView {
		return parseToNoteView(t, "```datachart\n"+`{"data":{"source":"url","url":"http://x","body":"same"}}`+"\n```\n")
	}
	a := mk()
	a.extractCharts()
	b := mk()
	b.extractCharts()
	require.Equal(t, a.Charts[0].Hash, b.Charts[0].Hash)
}

func TestExtractCharts_IgnoresNonChartBlocks(t *testing.T) {
	content := "```go\nfmt.Println(1)\n```\n\n```json\n{\"a\":1}\n```\n"
	n := parseToNoteView(t, content)
	n.extractCharts()
	require.Empty(t, n.Charts)
}

func TestExtractCharts_InvalidJSONSkippedWithWarning(t *testing.T) {
	content := "```datachart\nnot json at all\n```\n"
	n := parseToNoteView(t, content)
	n.extractCharts()
	require.Empty(t, n.Charts)
	require.NotEmpty(t, n.Warnings)
}

func TestExtractCharts_NoBlocks(t *testing.T) {
	n := parseToNoteView(t, "# Just a heading\n\nSome text.\n")
	n.extractCharts()
	require.Empty(t, n.Charts)
}
