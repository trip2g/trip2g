package enclavefix_test

import (
	"bytes"
	"testing"

	"trip2g/internal/enclavefix"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

func TestYouTubeEmbedStandalone(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(
			enclavefix.New(&enclavecore.Config{}),
		),
	)

	source := []byte(`![](https://www.youtube.com/watch?v=SJCGVbYN9XY)`)

	var buf bytes.Buffer
	err := md.Convert(source, &buf)
	require.NoError(t, err)

	html := buf.String()
	t.Logf("Generated HTML: %s", html)

	// Should contain YouTube embed
	require.Contains(t, html, "youtube", "Should contain youtube-related content")
}

func TestYouTubeShortLinkStandalone(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(
			enclavefix.New(&enclavecore.Config{}),
		),
	)

	source := []byte(`![](https://youtu.be/SJCGVbYN9XY)`)

	var buf bytes.Buffer
	err := md.Convert(source, &buf)
	require.NoError(t, err)

	html := buf.String()
	t.Logf("Generated HTML: %s", html)

	require.Contains(t, html, "youtube", "Should contain youtube-related content")
}
