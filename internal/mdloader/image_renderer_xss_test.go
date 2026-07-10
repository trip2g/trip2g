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

func TestYouTubeEmbedIDCannotInjectHTMLAttribute(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![](https://www.youtube.com/watch?v=video-id%22%20onload%3D%22trip2g-marker%22)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `<iframe`, "invalid YouTube IDs must not reach the embed template")
	require.NotContains(t, html, `onload="trip2g-marker"`, "YouTube IDs must not break out of iframe attributes")
	require.Contains(t, html, `&#34;`, "the rejected identifier must be escaped in the error message")
}

func TestTradingViewSymbolCannotInjectScript(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![](https://www.tradingview.com/chart/abc/?symbol=%22%3Balert(1)%3B%2F%2F)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `";alert(1)`, "TradingView symbols must not break out of widget JavaScript")
}

func TestBilibiliEmbedIDCannotInjectHTMLAttribute(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![](https://www.bilibili.com/video/%22%20onload%3D%22trip2g-marker%22)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `<iframe`, "invalid Bilibili IDs must not reach the embed template")
	require.NotContains(t, html, `onload="trip2g-marker"`)
}

func TestQuailLayoutCannotInjectHTMLAttribute(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![](https://quaily.com/list?layout=%22%20onload%3D%22trip2g-marker%22)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `<iframe`, "invalid Quail layouts must not reach the embed template")
	require.NotContains(t, html, `onload="trip2g-marker"`)
}

func TestDifyURLCannotInjectHTMLAttribute(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![](dify://udify.app/chatbot/%22%20onload%3D%22trip2g-marker%22)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `<iframe`, "invalid Dify URLs must not reach the embed template")
	require.NotContains(t, html, `onload="trip2g-marker"`)
}

func TestImageDimensionCannotInjectHTMLAttribute(t *testing.T) {
	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(`![image](https://example.com/image.jpg?w=%22%20onload%3D%22trip2g-marker%22)`),
		}},
		Log: &log,
	})
	require.NoError(t, err)
	html := string(pages.Map["/note"].HTML)
	require.NotContains(t, html, `onload="trip2g-marker"`)
	require.NotContains(t, html, `width=""`)
}
