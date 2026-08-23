package rendernotepage_test

import (
	"net/http"
	"strings"
	"testing"

	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func extractBetween(s, from, to string) string {
	start := strings.Index(s, from)
	if start == -1 {
		return ""
	}
	rest := s[start+len(from):]
	end := strings.Index(rest, to)
	if end == -1 {
		return ""
	}
	return rest[:end]
}

// The same [[note-b]] wikilink must resolve to the same href whether it
// stands in the page body or in the header note attached via the header:
// frontmatter field. Today the body is rendered with the request's domain
// host (DomainHTML["docs.example.com"], relative "/b") while the header note
// is looked up through templateviews.NVS, which wraps it without a domain
// host, so it renders DomainHTML[""] — the main-domain variant with the
// absolute https://docs.example.com/b.
func TestHeaderWikilink_ResolvesLikeBodyWikilink_OnCustomDomain(t *testing.T) {
	views := loadTestVault(t, []mdloader.SourceFile{
		{
			Path:    "site-nav.md",
			Content: []byte("---\nfree: true\n---\n- [[note-b]]"),
		},
		{
			Path:    "note-a.md",
			Content: []byte("---\nfree: true\nroute: docs.example.com/a\nheader: site-nav.md\n---\nBody link: [[note-b]]"),
		},
		{
			Path:    "note-b.md",
			Content: []byte("---\nfree: true\nroute: docs.example.com/b\n---\nPage B"),
		},
	})

	env, _, _ := cacheTestEnv(views, nil)

	ctx := newReqCtx(reqOpts{host: "docs.example.com", path: "/a"})
	runHandle(t, env, ctx, nil)
	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())

	page := string(body(t, ctx))

	content := extractBetween(page, `<div class="content__body`, `</div>`)
	require.Contains(t, content, `href="/b"`,
		"body wikilink must use the relative path on the note's own domain")

	nav := extractBetween(page, `<nav class="site-header__nav">`, `</nav>`)
	require.NotEmpty(t, nav, "site header nav must render from the header note's first list")
	require.NotContains(t, nav, `href="https://docs.example.com/b"`,
		"header wikilink must not use an absolute URL for the host the reader is already on")
	require.Contains(t, nav, `href="/b"`,
		"header wikilink must resolve exactly like the body wikilink")
}
