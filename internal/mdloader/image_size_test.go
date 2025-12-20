package mdloader_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// TestImageSizeWikilink tests that ![[image.jpg|20x20]] renders with width/height.
func TestImageSizeWikilink(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![[image.jpg|20x20]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// Should render with width and height attributes
	require.Contains(t, html, `width="20"`, "Should have width attribute")
	require.Contains(t, html, `height="20"`, "Should have height attribute")
	require.Contains(t, html, `<img src="image.jpg"`, "Should have img tag")
}

// TestImageSizeWikilinkWidthOnly tests that ![[image.jpg|100]] renders with width only.
func TestImageSizeWikilinkWidthOnly(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![[image.jpg|100]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// Should render with width attribute only
	require.Contains(t, html, `width="100"`, "Should have width attribute")
	require.NotContains(t, html, `height=`, "Should not have height attribute")
}

// TestImageSizeMarkdown tests that ![alt|20x20](url) renders with width/height.
func TestImageSizeMarkdown(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![arrow|20x20](tg_ce_5974249837439224721.webp)`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// Should render with width and height attributes
	require.Contains(t, html, `width="20"`, "Should have width attribute")
	require.Contains(t, html, `height="20"`, "Should have height attribute")
	require.Contains(t, html, `src="tg_ce_5974249837439224721.webp"`, "Should have correct src")
	// Alt should not contain the size specification
	require.Contains(t, html, `alt="arrow"`, "Alt should be just 'arrow' without size")
}

// TestImageSizeMarkdownEmoji tests that ![➡️|20x20](url) renders with width/height.
func TestImageSizeMarkdownEmoji(t *testing.T) {
	log := logger.TestLogger{}

	testCases := []struct {
		name    string
		content string
	}{
		{"arrow_text", `![arrow|20x20](image.webp)`},
		{"emoji_no_size", `![➡️](image.webp)`},
		{"emoji_with_size", `![➡️|20x20](image.webp)`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sourceFiles := []mdloader.SourceFile{{
				Path:    "note.md",
				Content: []byte(tc.content),
			}}

			pages, err := mdloader.Load(mdloader.Options{
				Sources: sourceFiles,
				Log:     &log,
			})
			require.NoError(t, err)

			html := string(pages.Map["/note"].HTML)
			t.Logf("Input: %s", tc.content)
			t.Logf("HTML: %s", html)

			require.Contains(t, html, `<img`, "Should have img tag")
		})
	}
}

// TestImageSizeMarkdownWidthOnly tests that ![alt|100](url) renders with width only.
func TestImageSizeMarkdownWidthOnly(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![arrow|100](image.png)`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// Should render with width attribute only
	require.Contains(t, html, `width="100"`, "Should have width attribute")
	require.NotContains(t, html, `height=`, "Should not have height attribute")
	require.Contains(t, html, `alt="arrow"`, "Alt should be just 'arrow' without size")
}

// TestImageSizeWikilinkWithAlt tests ![[image.jpg|alt text|20x20]] with both alt and size.
func TestImageSizeWikilinkWithAlt(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![[image.jpg|some description|50x30]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// Should render with width and height attributes
	require.Contains(t, html, `width="50"`, "Should have width attribute")
	require.Contains(t, html, `height="30"`, "Should have height attribute")
	require.Contains(t, html, `alt="some description"`, "Should have alt text without size")
}

// TestImageNoSize tests that images without size specification work normally.
func TestImageNoSize(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![[image.jpg]] and ![alt](other.png)`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// Should render without width/height attributes
	require.NotContains(t, html, `width=`, "Should not have width attribute")
	require.NotContains(t, html, `height=`, "Should not have height attribute")
	require.Contains(t, html, `<img src="image.jpg">`, "Should have img tag for wikilink")
	require.Contains(t, html, `src="other.png"`, "Should have img tag for markdown")
}

// TestImageSizeWithAssetReplace tests that size works with asset replacement.
func TestImageSizeWithAssetReplace(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![emoji|20x20](./assets/emoji.webp)`),
		Assets: map[string]*model.NoteAssetReplace{
			"./assets/emoji.webp": {
				ID:  1,
				URL: "http://example.com/emoji.webp",
			},
		},
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// Should have replaced URL with size attributes
	require.Contains(t, html, `src="http://example.com/emoji.webp"`, "Should have replaced URL")
	require.Contains(t, html, `width="20"`, "Should have width attribute")
	require.Contains(t, html, `height="20"`, "Should have height attribute")
}
