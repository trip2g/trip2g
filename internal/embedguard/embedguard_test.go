package embedguard_test

import (
	"testing"

	"trip2g/internal/embedguard"

	"github.com/stretchr/testify/require"
)

func denyAll(id int64) (bool, string) {
	return false, "[hidden]"
}

func TestWrapRoundTripKeepsAllowedContentWithoutMarkers(t *testing.T) {
	html := "<p>a</p>" + embedguard.Wrap(7, "<p>inner</p>") + "<p>b</p>"

	out := embedguard.Strip(html, func(id int64) (bool, string) { return true, "" })

	require.Equal(t, "<p>a</p><p>inner</p><p>b</p>", out)
}

func TestStripReplacesDeniedBlock(t *testing.T) {
	html := "<p>a</p>" + embedguard.Wrap(7, "<p>secret</p>") + "<p>b</p>"

	out := embedguard.Strip(html, denyAll)

	require.Equal(t, "<p>a</p>[hidden]<p>b</p>", out)
}

func TestStripDeniedOuterDropsAllowedInner(t *testing.T) {
	html := embedguard.Wrap(1, "outer"+embedguard.Wrap(2, "inner"))

	out := embedguard.Strip(html, func(id int64) (bool, string) {
		return id == 2, "[hidden]"
	})

	require.Equal(t, "[hidden]", out)
}

func TestStripDeniedInnerInsideAllowedOuter(t *testing.T) {
	html := embedguard.Wrap(1, "outer-"+embedguard.Wrap(2, "inner")+"-end")

	out := embedguard.Strip(html, func(id int64) (bool, string) {
		return id == 1, "[hidden]"
	})

	require.Equal(t, "outer-[hidden]-end", out)
}

func TestStripSiblingBlocksAreIndependent(t *testing.T) {
	html := embedguard.Wrap(1, "one") + embedguard.Wrap(2, "two") + embedguard.Wrap(1, "again")

	out := embedguard.Strip(html, func(id int64) (bool, string) {
		return id == 2, "[hidden]"
	})

	require.Equal(t, "[hidden]two[hidden]", out)
}

func TestStripUnterminatedDeniedBlockFailsClosed(t *testing.T) {
	html := "<p>a</p><!--t2e:5-->secret never closed"

	out := embedguard.Strip(html, denyAll)

	require.Equal(t, "<p>a</p>[hidden]", out)
}

func TestStripDropsStrayCloseMarker(t *testing.T) {
	out := embedguard.Strip("a<!--/t2e:5-->b", denyAll)

	require.Equal(t, "ab", out)
}

func TestStripLeavesPlainHTMLUntouched(t *testing.T) {
	html := "<p>a</p><!-- ordinary comment --><!--t2e:abc-->"

	require.Equal(t, html, embedguard.Strip(html, denyAll))
}

func TestHasMarkers(t *testing.T) {
	require.False(t, embedguard.HasMarkers("<p>a</p>"))
	require.True(t, embedguard.HasMarkers(embedguard.Wrap(1, "x")))
}
