package highlight_test

import (
	"bytes"
	"testing"
	"trip2g/internal/mdloader/highlight"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

func TestHighlight(t *testing.T) {
	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
		goldmark.WithExtensions(
			highlight.Highlight,
		),
	)

	source := []byte(`==hello==`)

	var buf bytes.Buffer

	err := md.Convert(source, &buf)
	require.NoError(t, err)

	require.Equal(t, "<p><mark>hello</mark></p>\n", buf.String())
}
