package enclavefix_test

import (
	"testing"

	"trip2g/internal/enclavefix"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func enclaves(t *testing.T, source string) []*enclavecore.Enclave {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(enclavefix.New(&enclavecore.Config{})))
	doc := md.Parser().Parse(text.NewReader([]byte(source)))

	var found []*enclavecore.Enclave
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if enc, ok := n.(*enclavecore.Enclave); ok && entering {
			found = append(found, enc)
		}
		return ast.WalkContinue, nil
	})
	return found
}

func TestTransformerProviders(t *testing.T) {
	tests := []struct {
		source   string
		provider string
		objectID string
	}{
		{"![](https://www.youtube.com/watch?v=SJCGVbYN9XY)", enclavecore.EnclaveProviderYouTube, "SJCGVbYN9XY"},
		{"![](https://youtu.be/SJCGVbYN9XY)", enclavecore.EnclaveProviderYouTube, "SJCGVbYN9XY"},
		{"![](https://x.com/user/status/1)", enclavecore.EnclaveProviderTwitter, "https://twitter.com/user/status/1"},
		{"![](https://udify.app/chatbot/abc)", enclavecore.EnclaveRegularImage, "https://udify.app/chatbot/abc"},
		{"![](https://quaily.com/list)", enclavecore.EnclaveRegularImage, "https://quaily.com/list"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			found := enclaves(t, tt.source)
			require.Len(t, found, 1)
			require.Equal(t, tt.provider, found[0].Provider)
			require.Equal(t, tt.objectID, found[0].ObjectID)
		})
	}
}

func TestEveryImageInParagraphIsTransformed(t *testing.T) {
	found := enclaves(t, "![](https://youtu.be/SJCGVbYN9XY) ![](https://youtu.be/dQw4w9WgXcQ)")
	require.Len(t, found, 2)
}

func TestValidEmbedIDPreservesSupportedIdentifiers(t *testing.T) {
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderYouTube, "dQw4w9WgXcQ"))
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderBilibili, "BV1xx411c7mD"))
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderTradingView, "CME_MINI:ES1!"))
	require.True(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderTradingView, "(NASDAQ:AAPL+NASDAQ:MSFT)/2"))
	require.False(t, enclavefix.ValidEmbedID(enclavecore.EnclaveProviderYouTube, `x" onload="y`))
}

func TestSafeImageDimension(t *testing.T) {
	require.Equal(t, "200", enclavefix.SafeImageDimension("200"))
	require.Equal(t, "50%", enclavefix.SafeImageDimension("50%"))
	require.Empty(t, enclavefix.SafeImageDimension(`200" onload="x`))
}
