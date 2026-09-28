package mdloader_test

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func renderOne(t *testing.T, content string) string {
	t.Helper()
	notes, err := mdloader.Load(mdloader.Options{
		Log:     &logger.TestLogger{},
		Sources: []mdloader.SourceFile{{Path: "a.md", Content: []byte(content)}},
	})
	require.NoError(t, err)
	return string(notes.PathMap["a.md"].HTML)
}

// Tweets render as X's standard client-side embed: no server-side fetch.
func TestTweetRendersClientSide(t *testing.T) {
	html := renderOne(t, "![](https://x.com/user/status/123)")
	require.Contains(t, html, `<blockquote class="twitter-tweet">`)
	require.Contains(t, html, `href="https://twitter.com/user/status/123"`)
	require.Contains(t, html, `src="https://platform.twitter.com/widgets.js"`)

	dark := renderOne(t, "![](https://twitter.com/user/status/123?theme=dark)")
	require.Contains(t, dark, `<blockquote class="twitter-tweet" data-theme="dark">`)
}

func TestTweetURLIsEscaped(t *testing.T) {
	html := renderOne(t, `![](https://twitter.com/user/status/1?a="><script>x</script>)`)
	require.NotContains(t, html, "<script>x")
}

func TestRemovedProvidersRenderAsPlainImages(t *testing.T) {
	for _, src := range []string{
		"https://udify.app/chatbot/1NaVTsaJ1t54UrNE",
		"https://quaily.com/list",
		"quaily://ads/123e4567-e89b-12d3-a456-426614174000",
	} {
		html := renderOne(t, "![]("+src+")")
		require.NotContains(t, html, "<iframe", src)
		require.NotContains(t, html, "<script", src)
	}
}
