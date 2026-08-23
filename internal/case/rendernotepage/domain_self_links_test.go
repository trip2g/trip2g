package rendernotepage_test

import (
	"net/http"
	"regexp"
	"testing"

	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

// A page served on host H must not contain absolute links back to H.
// The reader is already there: sending them away and back is a wasted round
// trip at best, and at worst a link to a name they cannot reach -- a staging
// copy, a preview deployment, a host reachable only from outside.
//
// Body and chrome are rendered through different paths, so this asserts the
// property of the whole response rather than of one element in it.
//
// Navigation only: rel=canonical, og:url and JSON-LD are absolute by their
// own specifications and are excluded on purpose.
func TestServedPage_HasNoAbsoluteLinksToItsOwnHost(t *testing.T) {
	const host = "docs.example.com"

	views := loadTestVault(t, []mdloader.SourceFile{
		{
			Path:    "site-nav.md",
			Content: []byte("---\nfree: true\n---\n- [[note-b]]\n- [[note-c]]"),
		},
		{
			Path:    "note-a.md",
			Content: []byte("---\nfree: true\nroute: " + host + "/a\nheader: site-nav.md\n---\nBody: [[note-b]] and [[note-c]]"),
		},
		{
			Path:    "note-b.md",
			Content: []byte("---\nfree: true\nroute: " + host + "/b\n---\nPage B"),
		},
		{
			Path:    "note-c.md",
			Content: []byte("---\nfree: true\nroute: " + host + "\n---\nPage C"),
		},
	})

	env, _, _ := cacheTestEnv(views, nil)

	ctx := newReqCtx(reqOpts{host: host, path: "/a"})
	runHandle(t, env, ctx, nil)
	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())

	page := string(body(t, ctx))

	selfAbsolute := regexp.MustCompile(`<a\b[^>]*href="https?://` + regexp.QuoteMeta(host) + `[/"]`)
	found := selfAbsolute.FindAllString(page, -1)
	require.Empty(t, found,
		"page served on %s links back to %s with an absolute URL: %v", host, host, found)
}
