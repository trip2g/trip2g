package enclavefix_test

import (
	"bytes"
	"testing"

	"trip2g/internal/enclavefix"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

func TestYouTubeVideoIDCannotInjectHTMLAttribute(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(
			enclavefix.New(&enclavecore.Config{}),
		),
	)

	// Once decoded as a query parameter, the video ID closes the generated
	// src attribute and attempts to add another HTML attribute.
	source := []byte(`![](https://www.youtube.com/watch?v=video-id%22%20onload%3D%22trip2g-marker%22%20data-x%3D%22)`)

	var buf bytes.Buffer
	require.NoError(t, md.Convert(source, &buf))

	require.NotContains(t, buf.String(), " onload=",
		"an untrusted YouTube video ID must remain data inside the src attribute")
}
