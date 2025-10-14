package mdloader_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

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
