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

func TestYouTubeObjectIDCannotBreakOutOfAttribute(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![](https://www.youtube.com/watch?v=%22%20onload%3D%22alert(1))`), &buf)
	require.NoError(t, err)
	html := buf.String()
	require.NotContains(t, html, `onload="alert`)
	require.NotContains(t, html, `src="https://www.youtube.com/embed/%22`)
	require.Contains(t, html, "Failed to load youtube")
}

func TestTradingViewSymbolCannotBreakOutOfScript(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![](https://www.tradingview.com/chart/abc/?symbol=%22%3Balert(1)%3B%2F%2F)`), &buf)
	require.NoError(t, err)
	html := buf.String()
	require.NotContains(t, html, `<script`)
	require.NotContains(t, html, `";alert`)
	require.NotContains(t, html, `alert(1)`, "rejected symbols are redacted, not echoed")
	require.Contains(t, html, "Failed to load tradingview")
}

func TestBilibiliObjectIDCannotBreakOutOfAttribute(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![](https://www.bilibili.com/video/%22%20onload%3D%22alert(1))`), &buf)
	require.NoError(t, err)
	html := buf.String()
	require.NotContains(t, html, `<iframe`)
	require.NotContains(t, html, `onload="alert(1)"`)
	require.Contains(t, html, "Failed to load bilibili")
}

func TestQuailLayoutCannotBreakOutOfAttribute(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![](https://quaily.com/list?layout=%22%20onload%3D%22alert(1))`), &buf)
	require.NoError(t, err)
	html := buf.String()
	require.NotContains(t, html, `<iframe`)
	require.NotContains(t, html, `onload="alert(1)"`)
	require.Contains(t, html, "Failed to load quail")
}

func TestDifyURLCannotBreakOutOfAttribute(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![](dify://udify.app/chatbot/%22%20onload%3D%22alert(1))`), &buf)
	require.NoError(t, err)
	html := buf.String()
	require.NotContains(t, html, `<iframe`)
	require.NotContains(t, html, `onload="alert(1)"`)
	require.Contains(t, html, "Failed to load dify")
}

func TestQuailImageAttributesAreEscaped(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![" onload="alert(1)](https://example.com/image.jpg?w=%22%20onload%3D%22alert(1))`), &buf)
	require.NoError(t, err)
	html := buf.String()
	require.NotContains(t, html, `onload="alert(1)"`)
	require.NotContains(t, html, `width=""`)
	require.Contains(t, html, `&#34;`)
}

func TestAudioURLCannotBreakOutOfAttribute(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![](<https://example.com/a.mp3?x=" onload="alert(1)>)`), &buf)
	require.NoError(t, err)
	html := buf.String()
	require.NotContains(t, html, `onload="alert(1)"`)
	require.Contains(t, html, `&#34;`)
}

func TestQuailImageUnitlessSizeUsesPixels(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))

	var buf bytes.Buffer
	err := md.Convert([]byte(`![image](https://example.com/image.jpg?w=200)`), &buf)
	require.NoError(t, err)
	require.Contains(t, buf.String(), `style="width: 200px;`)
}

func TestValidEmbedIDPreservesSupportedIdentifiers(t *testing.T) {
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderYouTube, "dQw4w9WgXcQ"))
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderBilibili, "BV1xx411c7mD"))
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderTradingView, "CME_MINI:ES1!"))
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderTradingView, "(NASDAQ:AAPL+NASDAQ:MSFT)/2"))
	require.True(t, enclavefix.ValidDifyURL("https://udify.app/chatbot/1NaVTsaJ1t54UrNE"))
}
