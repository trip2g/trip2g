package mdloader

import (
	"testing"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/text"

	"github.com/stretchr/testify/require"
)

// shouldWarnEmptyHTML decides whether empty rendered HTML is a real bug signal.
// Frontmatter-only notes (e.g. docs/_user_right_sidebar.md, a Jet-layout config
// note) are meant to render empty, so they must not warn.
func TestShouldWarnEmptyHTML(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(meta.Meta))

	frontmatterOnly := md.Parser().Parse(text.NewReader([]byte("---\ntitle: Sidebar\n---\n")))
	require.False(t, shouldWarnEmptyHTML(frontmatterOnly), "frontmatter-only note should not warn")

	withBody := md.Parser().Parse(text.NewReader([]byte("---\ntitle: Sidebar\n---\n\nHello world\n")))
	require.True(t, shouldWarnEmptyHTML(withBody), "a note with real body content that rendered empty is the genuine bug signature")

	require.False(t, shouldWarnEmptyHTML(nil), "nil doc should not warn")
}
