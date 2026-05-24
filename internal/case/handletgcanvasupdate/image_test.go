package handletgcanvasupdate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractFirstImage_Frontmatter(t *testing.T) {
	content := "---\ntitle: Test\nimage: cover.png\n---\n\nBody text here."
	img, body := extractFirstImage(content)
	require.Equal(t, "cover.png", img)
	require.Equal(t, "Body text here.", body)
}

func TestExtractFirstImage_Embed(t *testing.T) {
	content := "Some text\n![[photo.jpg]]\nMore text"
	img, body := extractFirstImage(content)
	require.Equal(t, "photo.jpg", img)
	require.Equal(t, "Some text\nMore text", body)
}

func TestExtractFirstImage_NonImageEmbed(t *testing.T) {
	content := "Text\n![[other.md]]\nMore"
	img, body := extractFirstImage(content)
	require.Empty(t, img)
	require.Equal(t, "Text\n![[other.md]]\nMore", body)
}

func TestExtractFirstImage_NoImage(t *testing.T) {
	content := "---\ntitle: Test\n---\n\nJust body."
	img, body := extractFirstImage(content)
	require.Empty(t, img)
	require.Equal(t, "Just body.", body)
}

func TestExtractFirstImage_FrontmatterWins(t *testing.T) {
	content := "---\nimage: fm.png\n---\n\n![[embed.jpg]]\nBody"
	img, body := extractFirstImage(content)
	require.Equal(t, "fm.png", img)
	// Embed is not stripped when frontmatter image wins
	require.Contains(t, body, "![[embed.jpg]]")
}
