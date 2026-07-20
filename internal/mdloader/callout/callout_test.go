package callout_test

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"trip2g/internal/mdloader/callout"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

func render(t *testing.T, source string) string {
	t.Helper()

	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
		goldmark.WithExtensions(
			callout.Extension,
		),
	)

	var buf bytes.Buffer
	require.NoError(t, md.Convert([]byte(source), &buf))
	return buf.String()
}

func TestBasicCallout(t *testing.T) {
	html := render(t, "> [!note]\n> some text\n")

	require.Contains(t, html, `class="callout callout--note"`)
	require.Contains(t, html, `class="callout__header"`)
	require.Contains(t, html, `class="callout__title"`)
	require.Contains(t, html, `class="callout__body"`)
	// Default title is the capitalized type.
	require.Contains(t, html, "Note")
	require.Contains(t, html, "some text")
	// Must be a div, not details.
	require.Contains(t, html, "<div ")
	require.NotContains(t, html, "<details")
}

func TestCustomTitle(t *testing.T) {
	html := render(t, "> [!warning] My Title\n> body\n")

	require.Contains(t, html, `class="callout callout--warning"`)
	require.Contains(t, html, "My Title")
	require.Contains(t, html, "body")
}

func TestCollapsedCallout(t *testing.T) {
	html := render(t, "> [!tip]-\n> hidden\n")

	require.Contains(t, html, "<details")
	require.Contains(t, html, `class="callout callout--tip"`)
	require.Contains(t, html, "<summary")
	// Collapsed: no open attribute.
	require.NotContains(t, html, "<details open")
	require.Contains(t, html, "hidden")
}

func TestExpandedCallout(t *testing.T) {
	html := render(t, "> [!tip]+\n> shown\n")

	require.Contains(t, html, "<details open")
	require.Contains(t, html, `class="callout callout--tip"`)
	require.Contains(t, html, "<summary")
	require.Contains(t, html, "shown")
}

func TestExpandedCalloutWithTitle(t *testing.T) {
	html := render(t, "> [!example]+ Custom\n> shown\n")

	require.Contains(t, html, "<details open")
	require.Contains(t, html, `class="callout callout--example"`)
	require.Contains(t, html, "Custom")
}

func TestUnknownType(t *testing.T) {
	html := render(t, "> [!custom]\n> body\n")

	require.Contains(t, html, `class="callout callout--custom"`)
	require.Contains(t, html, "Custom")
}

func TestBodyWithMarkup(t *testing.T) {
	html := render(t, "> [!note]\n> some **bold** text\n")

	require.Contains(t, html, `class="callout callout--note"`)
	require.Contains(t, html, "<strong>bold</strong>")
}

func TestTitleOnlyNoBody(t *testing.T) {
	html := render(t, "> [!info] Just a title\n")

	require.Contains(t, html, `class="callout callout--info"`)
	require.Contains(t, html, "Just a title")
	require.Contains(t, html, `class="callout__body"`)
}

func TestPlainBlockquoteNotCallout(t *testing.T) {
	html := render(t, "> normal quote\n> second line\n")

	require.Contains(t, html, "<blockquote>")
	require.NotContains(t, html, "callout")
}

func TestPlainBlockquoteWithBracketNotCallout(t *testing.T) {
	// Looks similar but missing the bang — should stay a blockquote.
	html := render(t, "> [note] plain text\n")

	require.Contains(t, html, "<blockquote>")
	require.NotContains(t, html, `class="callout`)
}

func TestCaseInsensitiveType(t *testing.T) {
	html := render(t, "> [!NOTE]\n> body\n")

	// Type is lowercased for the class.
	require.Contains(t, html, `class="callout callout--note"`)
	// Default title capitalizes the lowercased type.
	require.Contains(t, html, "Note")
}

func TestNonASCIITypeTitle(t *testing.T) {
	html := render(t, "> [!заметка]\n> body\n")

	// Default title capitalizes the first rune, not the first byte.
	require.Contains(t, html, "Заметка")
	require.True(t, utf8.ValidString(html))
}

func TestMultiLineBody(t *testing.T) {
	html := render(t, "> [!warning] Heads up\n> line one\n>\n> line two\n")

	require.Contains(t, html, `class="callout callout--warning"`)
	require.Contains(t, html, "Heads up")
	require.Contains(t, html, "line one")
	require.Contains(t, html, "line two")
}

func TestIconPresent(t *testing.T) {
	html := render(t, "> [!warning]\n> body\n")

	require.Contains(t, html, `class="callout__icon"`)
	// Warning icon.
	require.Contains(t, html, "⚠️")
}

func TestNestedBlockMarkupInBody(t *testing.T) {
	// A list inside the callout body should render through normal goldmark.
	html := render(t, "> [!info]\n> - one\n> - two\n")

	require.Contains(t, html, `class="callout callout--info"`)
	require.Contains(t, html, "<ul>")
	require.Contains(t, html, "<li>one</li>")
	require.Contains(t, html, "<li>two</li>")
}

func TestTitleIsHTMLEscaped(t *testing.T) {
	html := render(t, "> [!note] A & B <x>\n> body\n")

	require.Contains(t, html, "A &amp; B &lt;x&gt;")
	require.NotContains(t, html, "<x>")
}

func TestEmptyBodyStillRenders(t *testing.T) {
	html := render(t, "> [!note]\n")

	require.Contains(t, html, `class="callout callout--note"`)
	require.Contains(t, html, "Note")
	require.Contains(t, html, `class="callout__body"`)
}
