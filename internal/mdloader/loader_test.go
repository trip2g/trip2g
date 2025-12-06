package mdloader_test

import (
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/require"
)

func TestFlatIndexFirstSecond(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "index.md",
		Content: []byte(`Hello [[first]] [[second]]`),
	}, {
		Path: "first.md",
		Content: []byte(`---
title: First
---

First. Second [[second]] [[dead]]`),
	}, {
		Path:    "second.md",
		Content: []byte(`Second.`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)
	require.Len(t, pages.Map, 4)

	require.Equal(t, "index", pages.Map["/"].Title)
	require.Equal(t, "index", pages.Map["/index"].Title)
	require.Equal(t, "First", pages.Map["/first"].Title)
	require.Equal(t, "second", pages.Map["/second"].Title)

	require.Equal(t, map[string]struct{}{}, pages.Map["/index"].InLinks)
	require.Equal(t, map[string]struct{}{"/": {}}, pages.Map["/first"].InLinks)
	require.Equal(t, map[string]struct{}{"/": {}, "/first": {}}, pages.Map["/second"].InLinks)

	// Check if there's a warning about broken link
	hasBrokenLinkWarning := false
	for _, warning := range pages.Map["/first"].Warnings {
		if strings.Contains(warning.Message, "broken link") && strings.Contains(warning.Message, "dead") {
			hasBrokenLinkWarning = true
			break
		}
	}
	require.True(t, hasBrokenLinkWarning, "Expected warning about broken link to 'dead'")
}

func TestRelatedLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "second.md",
		Content: []byte(`---
free: true
---
Hello [[nested/first]]`),
	}, {
		Path: "nested/first.md",
		Content: []byte(`---
free: true
---
nested [[second]]`),
	}, {
		Path: "nested/second.md",
		Content: []byte(`---
free: true
---
nested second`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)
	require.Len(t, pages.Map, 3)

	// With Obsidian global resolution:
	// - [[second]] from /nested/first.md resolves to /second (root priority, not local /nested/second)
	// - [[nested/first]] from /second.md resolves to /nested/first (explicit path)
	require.Equal(t, map[string]struct{}{"/nested/first": {}}, pages.Map["/second"].InLinks)
	require.Equal(t, map[string]struct{}{"/second": {}}, pages.Map["/nested/first"].InLinks)
	require.Equal(t, map[string]struct{}{}, pages.Map["/nested/second"].InLinks)

	htmlSources := map[string]string{}

	for path, page := range pages.Map {
		htmlSources[path] = string(page.HTML)
	}

	cupaloy.SnapshotT(t, htmlSources)
}

func TestPaywallLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Hello [[Hidden]]`),
	}, {
		Path:    "Hidden.md",
		Content: []byte(`Payed content`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Config: mdloader.Config{
			AutoLowerWikilinks: true,
		},
	})
	require.NoError(t, err)

	htmlSources := map[string]string{}

	for path, page := range pages.Map {
		htmlSources[path] = string(page.HTML)
	}

	cupaloy.SnapshotT(t, htmlSources)
}

func TestRussianPaywallLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Hello [[Понедельник 9 июня 2025]]`),
	}, {
		Path:    "Понедельник 9 июня 2025.md",
		Content: []byte(`Payed content`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	htmlSources := map[string]string{}

	for path, page := range pages.Map {
		htmlSources[path] = string(page.HTML)
	}

	cupaloy.SnapshotT(t, htmlSources)
}

func TestRenamedPaywallLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "Понедельник 9 июня 2025.md",
		Content: []byte(`[[Шаблон дневной заметки|шаблона дня]]`),
	}, {
		Path:    "Шаблон дневной заметки.md",
		Content: []byte(`Content...`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	htmlSources := map[string]string{}

	for path, page := range pages.Map {
		htmlSources[path] = string(page.HTML)
	}

	cupaloy.SnapshotT(t, htmlSources)
}

func TestAssets(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "index.md",
		Content: []byte(`Hello ![[image.png]] and document [PDF](/file.pdf) and image ![hello](image2.png)`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	require.Equal(t, map[string]struct{}{
		"image.png":  struct{}{},
		"/file.pdf":  struct{}{},
		"image2.png": struct{}{},
	}, pages.Map["/index"].Assets)
}

func TestWIPLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Links: [[existing]] [[nonexistent]] [[another_missing]]`),
	}, {
		Path:    "existing.md",
		Content: []byte(`This page exists.`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	htmlSources := map[string]string{}

	for path, page := range pages.Map {
		htmlSources[path] = string(page.HTML)
	}

	cupaloy.SnapshotT(t, htmlSources)
}

// TestHardWraps tests that hard wraps are rendered correctly.
// Obsidian by default uses hard wraps when you press Enter,
// which means each line is a separate line in the markdown file,
// but they should be rendered as <br> tags in HTML.
func TestHardWraps(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "hard_wraps.md",
		Content: []byte(`This is a paragraph with hard wraps.
Obsidian by default uses hard wraps when you press Enter.
This means each line is a separate line in the markdown file.
But they should be rendered with <br> tags.

This is a new paragraph after an empty line.`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Config: mdloader.Config{
			SoftWraps: false, // Hard wraps (Obsidian default)
		},
	})
	require.NoError(t, err)

	html := string(pages.Map["/hard_wraps"].HTML)

	// Should contain <br> tags for hard wraps
	require.Contains(t, html, "<br>")

	// Should have two separate paragraphs
	require.Contains(t, html, "<p>This is a paragraph with hard wraps.")
	require.Contains(t, html, "<p>This is a new paragraph after an empty line.</p>")
}

// TestUniqueFilenameResolution tests that unique filenames resolve correctly
// even when they are in subdirectories.
// Bug: currently [[deep]] resolves to /deep instead of /folder/deep.
func TestUniqueFilenameResolution(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "unique.md",
		Content: []byte(`---
free: true
---
Link: [[deep]] - should find /folder/deep.md`),
	}, {
		Path: "folder/deep.md",
		Content: []byte(`---
free: true
---
Found me! Path: /folder/deep.md`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Check that the link resolves to /folder/deep
	uniquePage := pages.Map["/unique"]
	require.NotNil(t, uniquePage)

	// The link should be resolved to /folder/deep
	resolvedLink, found := uniquePage.ResolvedLinks["deep"]
	require.True(t, found, "Link 'deep' should be resolved to /folder/deep")
	require.Equal(t, "/folder/deep", resolvedLink, "Link should resolve to /folder/deep, not left unresolved")

	// Should not have broken link warning
	for _, warning := range uniquePage.Warnings {
		require.NotContains(t, warning.Message, "broken link: deep", "Should not have broken link warning for unique filename")
	}
}

// TestDuplicateFilenamesPriority tests Obsidian's critical behavior:
// When multiple files have the same name, links resolve to the file
// CLOSEST TO ROOT, not the file closest to the source.
func TestDuplicateFilenamesPriority(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "dup.md",
		Content: []byte(`# Root Duplicate
---
Found me! Path: /dup.md (root)`),
	}, {
		Path: "folder/dup.md",
		Content: []byte(`# Subfolder Duplicate
---
Found me! Path: /folder/dup.md`),
	}, {
		Path: "folder/source.md",
		Content: []byte(`# Source File
[[dup]]
---
This should link to /dup.md (root), NOT /folder/dup.md (local)!`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Check that the link resolves to root file, NOT the local one
	sourcePage := pages.Map["/folder/source"]
	require.NotNil(t, sourcePage)

	// The link should be resolved to /dup (root), not /folder/dup (local)
	resolvedLink, found := sourcePage.ResolvedLinks["dup"]
	require.True(t, found, "Link 'dup' should be resolved")
	require.Equal(t, "/dup", resolvedLink, "Link should resolve to /dup (root), not /folder/dup (local) - this is critical Obsidian behavior")

	// Should not have broken link warning
	for _, warning := range sourcePage.Warnings {
		require.NotContains(t, warning.Message, "broken link: dup", "Should not have broken link warning")
	}
}

// TestExplicitPathResolution tests that explicit paths like [[folder/file]]
// resolve to the specified path, not through global filename search.
func TestExplicitPathResolution(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "dup.md",
		Content: []byte(`# Root Duplicate
---
Found me! Path: /dup.md`),
	}, {
		Path: "folder/dup.md",
		Content: []byte(`# Subfolder Duplicate
---
Found me! Path: /folder/dup.md`),
	}, {
		Path: "source.md",
		Content: []byte(`# Source File
[[folder/dup]]
---
This should link to /folder/dup.md explicitly!`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Check that explicit path resolves correctly
	sourcePage := pages.Map["/source"]
	require.NotNil(t, sourcePage)

	// The link should be resolved to /folder/dup (explicit path)
	resolvedLink, found := sourcePage.ResolvedLinks["folder/dup"]
	require.True(t, found, "Link 'folder/dup' should be resolved")
	require.Equal(t, "/folder/dup", resolvedLink, "Explicit path should resolve to /folder/dup")

	// Should not have broken link warning
	for _, warning := range sourcePage.Warnings {
		require.NotContains(t, warning.Message, "broken link: folder/dup", "Should not have broken link warning")
	}
}

// TestCaseInsensitiveResolution tests that link resolution is case-insensitive
// matching Obsidian's behavior.
func TestCaseInsensitiveResolution(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "MyFile.md",
		Content: []byte(`# My File
---
Found me! Path: /MyFile.md`),
	}, {
		Path: "source.md",
		Content: []byte(`# Source File
[[myfile]]
[[MYFILE]]
[[MyFile]]
---
All three should resolve to /MyFile.md`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Check all case variations resolve
	sourcePage := pages.Map["/source"]
	require.NotNil(t, sourcePage)

	// All case variations should resolve to /myfile (normalized to lowercase)
	for _, linkText := range []string{"myfile", "MYFILE", "MyFile"} {
		resolvedLink, found := sourcePage.ResolvedLinks[linkText]
		require.True(t, found, "Link '%s' should be resolved (case-insensitive)", linkText)
		require.Equal(t, "/myfile", resolvedLink, "Link '%s' should resolve to /myfile (normalized)", linkText)
	}

	// Should not have broken link warnings
	for _, warning := range sourcePage.Warnings {
		require.NotContains(t, warning.Message, "broken link:", "Should not have any broken link warnings")
	}
}

// TestRelativePathResolution tests that explicit relative paths like [[./file]]
// and [[../file]] resolve correctly relative to the source file's location.
func TestRelativePathResolution(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "dup.md",
		Content: []byte(`# Root Duplicate
---
Found me! Path: /dup.md`),
	}, {
		Path: "folder/dup.md",
		Content: []byte(`# Subfolder Duplicate
---
Found me! Path: /folder/dup.md`),
	}, {
		Path: "folder/source.md",
		Content: []byte(`# Source File
[[dup]] - goes to root
[[./dup]] - stays local
[[folder/dup]] - explicit path from root
---
Testing relative path resolution`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Check source page
	sourcePage := pages.Map["/folder/source"]
	require.NotNil(t, sourcePage)

	// [[dup]] should resolve to /dup (root, via global resolution)
	resolvedLink1, found1 := sourcePage.ResolvedLinks["dup"]
	require.True(t, found1, "Link 'dup' should be resolved")
	require.Equal(t, "/dup", resolvedLink1, "[[dup]] should resolve to /dup (root)")

	// [[./dup]] should resolve to /folder/dup (local, relative path)
	resolvedLink2, found2 := sourcePage.ResolvedLinks["./dup"]
	require.True(t, found2, "Link './dup' should be resolved")
	require.Equal(t, "/folder/dup", resolvedLink2, "[[./dup]] should resolve to /folder/dup (local)")

	// [[folder/dup]] should resolve to /folder/dup (explicit path)
	resolvedLink3, found3 := sourcePage.ResolvedLinks["folder/dup"]
	require.True(t, found3, "Link 'folder/dup' should be resolved")
	require.Equal(t, "/folder/dup", resolvedLink3, "[[folder/dup]] should resolve to /folder/dup")

	// Should not have broken link warnings
	for _, warning := range sourcePage.Warnings {
		require.NotContains(t, warning.Message, "broken link:", "Should not have any broken link warnings")
	}
}

// TestSoftWraps tests that soft wraps are rendered correctly.
// With soft wraps enabled, consecutive lines without empty lines
// should be rendered as a single paragraph without <br> tags.
func TestSoftWraps(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "soft_wraps.md",
		Content: []byte(`This is a paragraph with soft wraps.
These lines should be combined.
Into a single paragraph.
Without line breaks.

This is a new paragraph after an empty line.`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Config: mdloader.Config{
			SoftWraps: true, // Soft wraps
		},
	})
	require.NoError(t, err)

	html := string(pages.Map["/soft_wraps"].HTML)

	// Should NOT contain <br> tags for soft wraps
	require.NotContains(t, html, "<br>")

	// Should have two separate paragraphs
	require.Contains(t, html, "<p>This is a paragraph with soft wraps.")
	require.Contains(t, html, "<p>This is a new paragraph after an empty line.</p>")
}

// TestVideoAssets tests that video files (.mp4, .webm, etc.) are correctly detected as assets.
func TestVideoAssets(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "media_group.md",
		Content: []byte(`Media content: ![[video.mp4]] and ![[photo.png]] and ![[clip.webm]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// All media files should be detected as assets
	require.Equal(t, map[string]struct{}{
		"video.mp4": struct{}{},
		"photo.png": struct{}{},
		"clip.webm": struct{}{},
	}, pages.Map["/media_group"].Assets)
}

// TestExternalURLsNotAssets tests that external URLs (http://, https://) are NOT marked as assets.
func TestExternalURLsNotAssets(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "links.md",
		Content: []byte(`Links: [Google](https://google.com) and [Local PDF](file.pdf) and [External](http://example.com/doc.pdf)

Image: ![alt](local.png) and remote ![remote](https://example.com/image.png)`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Only local files should be assets, NOT external URLs
	require.Equal(t, map[string]struct{}{
		"file.pdf":  struct{}{},
		"local.png": struct{}{},
	}, pages.Map["/links"].Assets)
}

// TestVideoRendering tests that video files are rendered as <video> tags, not <img>.
func TestVideoRendering(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "media.md",
		Content: []byte(`Image: ![[photo.png]]

Video: ![[clip.mp4]]

Another video: ![[movie.webm]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/media"].HTML)

	// Images should use <img> tag
	require.Contains(t, html, `<img src="photo.png">`)

	// Videos should use <video> tag with controls
	require.Contains(t, html, `<video controls src="clip.mp4">`)
	require.Contains(t, html, `<video controls src="movie.webm">`)

	// Videos should NOT use <img> tag
	require.NotContains(t, html, `<img src="clip.mp4"`)
	require.NotContains(t, html, `<img src="movie.webm"`)
}

// TestVersionedLinks tests that links with version parameter preserve slashes in paths.
// Bug: url.PathEscape encodes "/" as "%2F", breaking paths like "/folder/source".
func TestVersionedLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Links: [[folder/source]] [[simple]]`),
	}, {
		Path: "folder/source.md",
		Content: []byte(`---
free: true
---
Content in folder`),
	}, {
		Path: "simple.md",
		Content: []byte(`---
free: true
---
Simple content`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "v1.2.3",
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Path slashes should NOT be encoded as %2F
	require.NotContains(t, html, `%2F`, "Slashes in paths should not be URL-encoded")

	// Version parameter should be present
	require.Contains(t, html, `?version=v1.2.3`, "Version parameter should be in URL")

	// The href should look like: href="/folder/source?version=v1.2.3"
	require.Contains(t, html, `href="/folder/source?version=v1.2.3"`, "Path with version should preserve slashes")
	require.Contains(t, html, `href="/simple?version=v1.2.3"`, "Simple path with version should work")
}

// TestVersionedLinksWithSpecialChars tests that paths with special characters
// are normalized (transliterated) and slashes are preserved.
func TestVersionedLinksWithSpecialChars(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Links: [[100% силы]] [[путь/файл]]`),
	}, {
		Path: "100% силы.md",
		Content: []byte(`---
free: true
---
Content with percent`),
	}, {
		Path: "путь/файл.md",
		Content: []byte(`---
free: true
---
Content in Cyrillic path`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Path slashes should NOT be encoded as %2F
	require.NotContains(t, html, `%2F`, "Slashes should not be encoded")

	// Cyrillic paths are transliterated, slashes preserved
	// "путь/файл" becomes "/putj/fajl" with slash preserved
	require.Contains(t, html, `/putj/fajl?version=latest`, "Cyrillic path should be transliterated with slashes preserved")

	// Version parameter should be present
	require.Contains(t, html, `?version=latest`, "Version parameter should be in URL")
}

// TestVersionedLinksNotAppliedToImages tests that version parameter is NOT added to image links.
func TestVersionedLinksNotAppliedToImages(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Image: ![[photo.png]]
Link: [[page]]`),
	}, {
		Path: "page.md",
		Content: []byte(`---
free: true
---
Page content`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "v1.0",
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Images should NOT have version parameter
	require.Contains(t, html, `<img src="photo.png">`, "Image should not have version parameter")
	require.NotContains(t, html, `photo.png?version`, "Image should not have version in URL")

	// Links should have version parameter
	require.Contains(t, html, `href="/page?version=v1.0"`, "Link should have version parameter")
}

// TestDefaultVersionNoParameter tests that default "live" version doesn't add ?version= parameter.
func TestDefaultVersionNoParameter(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Link: [[page]]`),
	}, {
		Path: "page.md",
		Content: []byte(`---
free: true
---
Page content`),
	}}

	// Test with default "live" version
	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "live",
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Should NOT have version parameter for default version
	require.NotContains(t, html, `?version=`, "Default 'live' version should not add version parameter")
	require.Contains(t, html, `href="/page"`, "Link should not have version parameter")
}

// TestSlugWithCyrillicNoDoubleEncoding tests that pages with cyrillic slug
// don't get double URL-encoded when linked from other pages.
// Bug: link.Target was set to URL-encoded Permalink, then link_renderer
// encoded it again via util.URLEscape, resulting in %25D0%25... instead of %D0%...
func TestSlugWithCyrillicNoDoubleEncoding(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Link to cyrillic slug: [[slug_cyrillic]]`),
	}, {
		Path: "slug_cyrillic.md",
		Content: []byte(`---
free: true
slug: моя-страница
title: Cyrillic Slug Page
---
Content with cyrillic slug`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Should have properly encoded cyrillic URL (single encoding)
	// %D0%BC%D0%BE%D1%8F = "моя" in URL encoding
	require.Contains(t, html, `href="/%D0%BC%D0%BE%D1%8F-%D1%81%D1%82%D1%80%D0%B0%D0%BD%D0%B8%D1%86%D0%B0?version=latest"`,
		"Cyrillic slug should be URL-encoded once, not double-encoded")

	// Should NOT have double-encoded percent signs (%25)
	require.NotContains(t, html, `%25`,
		"Should not have double-encoded percent signs")
}

// TestSlugWithSpacesNoDoubleEncoding tests that pages with spaces in slug
// don't get double URL-encoded when linked from other pages.
func TestSlugWithSpacesNoDoubleEncoding(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Link to slug with spaces: [[slug_spaces]]`),
	}, {
		Path: "slug_spaces.md",
		Content: []byte(`---
free: true
slug: page with spaces
title: Page With Spaces
---
Content with spaces in slug`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Should have properly encoded space as %20 (single encoding)
	require.Contains(t, html, `href="/page%20with%20spaces?version=latest"`,
		"Spaces in slug should be URL-encoded as %%20, not double-encoded")

	// Should NOT have double-encoded %2520 (where %25 is encoded %)
	require.NotContains(t, html, `%2520`,
		"Should not have double-encoded spaces (%%2520)")
}

// TestSlugLinksNotMarkedAsWIP tests that pages with custom slug are found
// by link_renderer and not marked as WIP (class="wip").
func TestSlugLinksNotMarkedAsWIP(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
[[slug_cyrillic]] [[slug_spaces]]`),
	}, {
		Path: "slug_cyrillic.md",
		Content: []byte(`---
free: true
slug: моя-страница
---
Cyrillic slug`),
	}, {
		Path: "slug_spaces.md",
		Content: []byte(`---
free: true
slug: page with spaces
---
Spaces slug`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Links should NOT have class="wip" since the pages exist
	require.NotContains(t, html, `class="wip"`,
		"Links to existing pages with custom slug should not be marked as WIP")

	// Links should have data-pid attribute (proving page was found)
	require.Contains(t, html, `data-pid=`,
		"Links should have data-pid attribute since pages exist")

	// Should NOT have double-encoded URLs
	require.NotContains(t, html, `%25`,
		"Should not have double-encoded percent signs")
	require.NotContains(t, html, `%2520`,
		"Should not have double-encoded spaces")
}

// TestEmptyVersionNoParameter tests that empty version doesn't add ?version= parameter.
func TestEmptyVersionNoParameter(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Link: [[page]]`),
	}, {
		Path: "page.md",
		Content: []byte(`---
free: true
---
Page content`),
	}}

	// Test with empty version
	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "",
	})
	require.NoError(t, err)

	html := string(pages.Map["/"].HTML)

	// Should NOT have version parameter for empty version
	require.NotContains(t, html, `?version=`, "Empty version should not add version parameter")
	require.Contains(t, html, `href="/page"`, "Link should not have version parameter")
}
