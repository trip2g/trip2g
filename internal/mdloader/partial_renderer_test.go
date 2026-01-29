package mdloader_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestPartialLoader(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`
Hello

## Header 2

content

### Header 3

content

## Header 2

content
`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)
	require.Len(t, pages.List, 1)

	nv := pages.List[0]

	// First call
	blocks := nv.PartialRenderer.HeadingBlocks(2)
	require.Len(t, blocks, 2)

	// Second call - should return the same results
	blocks2 := nv.PartialRenderer.HeadingBlocks(2)
	require.Len(t, blocks2, 2)

	// Check that content is still there in both calls
	require.NotEmpty(t, blocks[0].ContentHTML)
	require.NotEmpty(t, blocks2[0].ContentHTML)
	require.Equal(t, blocks[0].ContentHTML, blocks2[0].ContentHTML)

	// Check that TitleHTML doesn't contain the heading tag itself
	require.NotContains(t, blocks[0].TitleHTML, "<h2>")
	require.NotContains(t, blocks[0].TitleHTML, "</h2>")
}

func TestPartialLoaderTitleContent(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`
## *Italic* and **Bold** Title

Some content here

## Another Title

More content
`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)
	require.Len(t, pages.List, 1)

	nv := pages.List[0]
	blocks := nv.PartialRenderer.HeadingBlocks(2)
	require.Len(t, blocks, 2)

	// Check that TitleHTML contains only the inner content
	require.Contains(t, blocks[0].TitleHTML, "<em>")
	require.Contains(t, blocks[0].TitleHTML, "<strong>")
	require.NotContains(t, blocks[0].TitleHTML, "<h2>")
	require.NotContains(t, blocks[0].TitleHTML, "</h2>")
}

func TestPartialLoaderIntroduce(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`
Welcome to our application! This is an **introduction** paragraph.

Another paragraph with *some* formatting.

## First Heading

Content after first heading.

### Subheading

More content here.
`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)
	require.Len(t, pages.List, 1)

	nv := pages.List[0]
	intro := nv.PartialRenderer.Introduce()

	// Check that TitleHTML is empty (no title for introduction)
	require.Empty(t, intro.TitleHTML)

	// Check that ContentHTML contains the introduction paragraphs
	require.Contains(t, intro.ContentHTML, "Welcome to our application!")
	require.Contains(t, intro.ContentHTML, "<strong>introduction</strong>")
	require.Contains(t, intro.ContentHTML, "<em>some</em>")
	require.Contains(t, intro.ContentHTML, "Another paragraph")

	// Check that it doesn't contain content after headings
	require.NotContains(t, intro.ContentHTML, "Content after first heading")
	require.NotContains(t, intro.ContentHTML, "More content here")
	require.NotContains(t, intro.ContentHTML, "<h2>")
}

func TestPartialLoaderIntroduceNoHeadings(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`
This is content without any headings.

Just paragraphs and **formatting**.
`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)
	require.Len(t, pages.List, 1)

	nv := pages.List[0]
	intro := nv.PartialRenderer.Introduce()

	// Check that TitleHTML is empty
	require.Empty(t, intro.TitleHTML)

	// Check that ContentHTML contains all content when no headings
	require.Contains(t, intro.ContentHTML, "content without any headings")
	require.Contains(t, intro.ContentHTML, "<strong>formatting</strong>")
}

// TestPartialRendererImageAssetReplace verifies that images inside sections
// get their URLs resolved from AssetReplaces, not left as relative paths.
func TestPartialRendererImageAssetReplace(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "slides.md",
		Content: []byte(`
## Category 1

![photo](./assets/photo.jpg)

### Slide 1

![slide-img](./assets/slide1.png)
`),
		Assets: map[string]*model.NoteAssetReplace{
			"./assets/photo.jpg": {
				ID:  1,
				URL: "http://cdn.example.com/photo-resolved.jpg",
			},
			"./assets/slide1.png": {
				ID:  2,
				URL: "http://cdn.example.com/slide1-resolved.png",
			},
		},
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)
	require.Len(t, pages.List, 1)

	nv := pages.List[0]

	// Full HTML should resolve images (sanity check).
	fullHTML := string(nv.HTML)
	require.Contains(t, fullHTML, "cdn.example.com/photo-resolved.jpg")
	require.Contains(t, fullHTML, "cdn.example.com/slide1-resolved.png")

	// Sections(2) ContentHTML should also resolve images.
	sections := nv.PartialRenderer.Sections(2)
	require.Len(t, sections, 1)
	require.Contains(t, sections[0].ContentHTML, "cdn.example.com/photo-resolved.jpg",
		"image in section ContentHTML should use resolved URL")

	// Nested Sections(3) ContentHTML should also resolve images.
	slides := sections[0].Sections(3)
	require.Len(t, slides, 1)
	require.Contains(t, slides[0].ContentHTML, "cdn.example.com/slide1-resolved.png",
		"image in nested section ContentHTML should use resolved URL")

	// Original relative paths should NOT appear.
	require.NotContains(t, sections[0].ContentHTML, "./assets/photo.jpg")
	require.NotContains(t, slides[0].ContentHTML, "./assets/slide1.png")
}
