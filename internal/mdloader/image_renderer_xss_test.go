package mdloader_test

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

// TestImageAltXSS verifies that a double-quote in alt text cannot break out of the attribute.
func TestImageAltXSS(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![x" onerror="alert(1)](image.png)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	// The quote must be escaped to &#34; so onerror cannot break out of the attribute.
	require.NotContains(t, html, `alt="x" onerror`, "unescaped quote in alt must not allow attribute breakout")
	require.Contains(t, html, `&#34;`, "double-quote in alt must be HTML-escaped")
}

// TestImageAltXSSWithSize verifies escaping when size is also present in alt.
func TestImageAltXSSWithSize(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![x" onerror="x|20x20](image.png)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `alt="x" onerror`, "unescaped quote in alt with size must not allow attribute breakout")
	require.Contains(t, html, `&#34;`, "double-quote in alt must be HTML-escaped")
}

// TestImageSrcJavascriptScheme verifies that javascript: URLs are rejected in img src.
func TestImageSrcJavascriptScheme(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![alt](javascript:alert(1))`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `src="javascript:`, "javascript: URLs must not appear in img src")
}

// TestImageSrcDataScheme verifies that data: URLs are rejected in img src.
func TestImageSrcDataScheme(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![alt](data:text/html,<script>alert(1)</script>)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `src="data:text/html`, "data:text/html URLs must not appear in img src")
}

// TestImageAltSpecialCharsEscaped verifies <, >, & in alt are HTML-escaped.
func TestImageAltSpecialCharsEscaped(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![<script>x</script>](image.png)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `<script>`, "script tags in alt must be escaped")
}
