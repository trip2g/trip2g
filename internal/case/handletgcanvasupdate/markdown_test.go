package handletgcanvasupdate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderBodyHTML_PlainText(t *testing.T) {
	got := renderBodyHTML("Hello world")
	require.Equal(t, "Hello world", got)
}

func TestRenderBodyHTML_HTMLEscaping(t *testing.T) {
	got := renderBodyHTML("a < b & c > d")
	require.Equal(t, "a &lt; b &amp; c &gt; d", got)
}

func TestRenderBodyHTML_Headings(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"h1", "# Title", "<b>TITLE</b>"},
		{"h2", "## Section", "<b>Section</b>"},
		{"h3", "### Sub", "<b><i>Sub</i></b>"},
		{"h4", "#### Deep", "<b><i>Deep</i></b>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBodyHTML(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRenderBodyHTML_CodeFence(t *testing.T) {
	input := "before\n```go\nfunc main() {}\n```\nafter"
	got := renderBodyHTML(input)
	require.Contains(t, got, "<pre>")
	require.Contains(t, got, "func main() {}")
	require.Contains(t, got, "</pre>")
	require.Contains(t, got, "before")
	require.Contains(t, got, "after")
	// Language hint stripped
	require.NotContains(t, got, "go\n")
}

func TestRenderBodyHTML_InlineMarkdown(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bold", "**bold**", "<b>bold</b>"},
		{"italic", "*italic*", "<i>italic</i>"},
		{"code", "`code`", "<code>code</code>"},
		{"link", "[text](http://example.com)", `<a href="http://example.com">text</a>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBodyHTML(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRenderLineHTML_NotAHeading(t *testing.T) {
	// # without space after it
	got := renderLineHTML("#nospace")
	require.Equal(t, "#nospace", got)
}

func TestApplyInlineMarkdown_Order(t *testing.T) {
	// Inline code regex fires first, but the bold inside is still processed
	// because the regex replaces the backtick-delimited span, then bold runs
	// on the result. This matches the demo behavior — a known limitation.
	got := applyInlineMarkdown("`**not bold**`")
	require.Equal(t, "<code><b>not bold</b></code>", got)
}

// TestApplyInlineMarkdown_LinkUrlEscaping is the regression test for the
// double-escape bug architect found: previously applyInlineMarkdown received
// already-HTML-escaped text, so an `&` inside a link URL became `&amp;` before
// the link regex ran and was emitted into href verbatim (still `&amp;`),
// breaking the URL. Now links are extracted from raw text first.
func TestApplyInlineMarkdown_LinkUrlEscaping(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "ampersand in URL stays single-escaped",
			in:   "see [docs](https://example.com?a=1&b=2)",
			want: `see <a href="https://example.com?a=1&amp;b=2">docs</a>`,
		},
		{
			name: "angle in URL escaped exactly once",
			in:   "go [home](https://example.com/<id>)",
			want: `go <a href="https://example.com/&lt;id&gt;">home</a>`,
		},
		{
			name: "link adjacent to bold",
			in:   "**bold** and [link](https://x.y)",
			want: `<b>bold</b> and <a href="https://x.y">link</a>`,
		},
		{
			name: "no link still escapes surrounding text",
			in:   "a & b",
			want: "a &amp; b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, applyInlineMarkdown(tt.in))
		})
	}
}

func TestHtmlEscape(t *testing.T) {
	require.Equal(t, "&amp;&lt;&gt;", htmlEscape("&<>"))
}
