package mdloader_test

import (
	"strings"
	"testing"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

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

// TestNavigationLinksNotAssets tests that navigation links like [Home](/) are NOT treated as assets.
func TestNavigationLinksNotAssets(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "sidebar.md",
		Content: []byte(`Navigation:
- [Home](/)
- [About](/about)
- [Public](/public)

Image: ![alt](image.png)`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Only media files should be assets, NOT navigation links
	require.Equal(t, map[string]struct{}{
		"image.png": struct{}{},
	}, pages.Map["/sidebar"].Assets)
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

	// Version parameter should NOT be present in links (removed in favor of session-based version)
	require.NotContains(t, html, `?version=`, "Version parameter should not be in links")

	// The href should look like: href="/folder/source"
	require.Contains(t, html, `href="/folder/source"`, "Path should preserve slashes")
	require.Contains(t, html, `href="/simple"`, "Simple path should work")
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
	require.Contains(t, html, `href="/putj/fajl"`, "Cyrillic path should be transliterated with slashes preserved")

	// Version parameter should NOT be present in links
	require.NotContains(t, html, `?version=`, "Version parameter should not be in links")
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

	// Links should NOT have version parameter (version is session-based now)
	require.Contains(t, html, `href="/page"`, "Link should not have version parameter")
	require.NotContains(t, html, `?version=`, "Version parameter should not be in links")
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
	require.Contains(t, html, `href="/%D0%BC%D0%BE%D1%8F-%D1%81%D1%82%D1%80%D0%B0%D0%BD%D0%B8%D1%86%D0%B0"`,
		"Cyrillic slug should be URL-encoded once, not double-encoded")

	// Should NOT have double-encoded percent signs (%25)
	require.NotContains(t, html, `%25`,
		"Should not have double-encoded percent signs")

	// Should NOT have version parameter
	require.NotContains(t, html, `?version=`, "Version parameter should not be in links")
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
	require.Contains(t, html, `href="/page%20with%20spaces"`,
		"Spaces in slug should be URL-encoded as %20, not double-encoded")

	// Should NOT have double-encoded %2520 (where %25 is encoded %)
	require.NotContains(t, html, `%2520`,
		"Should not have double-encoded spaces (%2520)")

	// Should NOT have version parameter
	require.NotContains(t, html, `?version=`, "Version parameter should not be in links")
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

// TestAllPagesHaveHTMLWithVersion tests that all pages have non-empty HTML
// when using versioned links. This is a regression test for a bug where
// the escapePathPreserveSlashes function caused HTML to be empty.
func TestAllPagesHaveHTMLWithVersion(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Links: [[page1]] [[folder/page2]] [[page3]]`),
	}, {
		Path: "page1.md",
		Content: []byte(`---
free: true
---
Page 1 content with link to [[page3]]`),
	}, {
		Path: "folder/page2.md",
		Content: []byte(`---
free: true
---
Page 2 in folder`),
	}, {
		Path: "page3.md",
		Content: []byte(`---
free: true
---
Page 3 content`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	// All pages must have non-empty HTML
	for path, page := range pages.PathMap {
		require.NotEmpty(t, page.HTML, "Page %s should have non-empty HTML", path)
		require.NotEmpty(t, page.Content, "Page %s should have non-empty Content", path)
	}
}

// TestEmbedOrderDoesNotAffectHTML tests that pages with embeds get HTML
// regardless of the order they are processed. This is a regression test
// for a bug where embed dependencies could cause empty HTML.
func TestEmbedOrderDoesNotAffectHTML(t *testing.T) {
	log := logger.TestLogger{}

	// Create a scenario where index embeds software, which embeds scenarios
	sourceFiles := []mdloader.SourceFile{{
		Path: "weekly_digest/index.md",
		Content: []byte(`---
free: true
---
Weekly digest with embed:
![[software]]`),
	}, {
		Path: "software.md",
		Content: []byte(`---
free: true
---
Software page with embed:
![[_scenarios]]`),
	}, {
		Path: "_scenarios.md",
		Content: []byte(`---
free: true
---
Scenarios content here`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	// All pages must have non-empty HTML
	for path, page := range pages.PathMap {
		require.NotEmpty(t, page.HTML, "Page %s should have non-empty HTML", path)
		require.NotEmpty(t, page.Content, "Page %s should have non-empty Content", path)
	}

	// Specifically check software.md has HTML
	softwarePage := pages.PathMap["software.md"]
	require.NotNil(t, softwarePage, "software.md should exist")
	require.NotEmpty(t, softwarePage.HTML, "software.md should have non-empty HTML")
	require.Contains(t, string(softwarePage.HTML), "Scenarios content", "software.md should contain embedded scenarios")
}

// TestPagesWithEmptyContentDontBreakOthers tests that pages with empty content
// don't break HTML generation for other pages. This is a regression test for
// a bug where escapePathPreserveSlashes caused issues when empty content pages existed.
func TestPagesWithEmptyContentDontBreakOthers(t *testing.T) {
	log := logger.TestLogger{}

	// Mix of pages with content and empty pages (simulating DB with empty content)
	sourceFiles := []mdloader.SourceFile{{
		Path: "index.md",
		Content: []byte(`---
free: true
---
Links: [[page1]] [[empty1]] [[page2]]`),
	}, {
		Path: "page1.md",
		Content: []byte(`---
free: true
---
Page 1 content`),
	}, {
		Path:    "empty1.md",
		Content: []byte(``), // Empty content - simulates DB issue
	}, {
		Path:    "empty2.md",
		Content: []byte(``), // Another empty page
	}, {
		Path: "page2.md",
		Content: []byte(`---
free: true
---
Page 2 content with link [[page1]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	// Pages with content must have non-empty HTML
	for _, path := range []string{"index.md", "page1.md", "page2.md"} {
		page := pages.PathMap[path]
		require.NotNil(t, page, "Page %s should exist", path)
		require.NotEmpty(t, page.HTML, "Page %s should have non-empty HTML", path)
	}

	// Empty pages should have empty HTML (not crash)
	for _, path := range []string{"empty1.md", "empty2.md"} {
		page := pages.PathMap[path]
		require.NotNil(t, page, "Page %s should exist", path)
		// Empty content = empty HTML, that's OK
	}
}

// TestEmbedWithCyrillicLinks tests that links inside embedded notes
// are correctly resolved to existing pages.
// Bug: Links in embedded notes were marked as "wip" even though target pages exist.
func TestEmbedWithCyrillicLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "main.md",
		Content: []byte(`---
free: true
---
![[_embed]]`),
	}, {
		Path:    "_embed.md",
		Content: []byte(`[[Кириллица]]`),
	}, {
		Path: "Кириллица.md",
		Content: []byte(`---
free: true
---
Cyrillic page content`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	// Main page should have non-empty HTML with embedded content
	mainPage := pages.PathMap["main.md"]
	require.NotNil(t, mainPage, "main.md should exist")
	require.NotEmpty(t, mainPage.HTML, "main.md should have non-empty HTML")

	// The embedded link should NOT be marked as wip
	require.NotContains(t, string(mainPage.HTML), `class="wip"`,
		"Link in embed should not be marked as wip")

	// The link should have data-pid (proving page was found)
	require.Contains(t, string(mainPage.HTML), `data-pid=`,
		"Link should have data-pid since target page exists")
}

// TestLinkWithDotInName tests that links to pages with dots in filename
// are correctly resolved.
// Bug: filepath.Ext("Сценарий. Ютубер") returns ". Ютубер" treating it as extension,
// so basename lookup fails because index key != search key.
func TestLinkWithDotInName(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "main.md",
		Content: []byte(`---
free: true
---
![[_embed]]`),
	}, {
		Path:    "_embed.md",
		Content: []byte(`[[Сценарий. Ютубер]]`),
	}, {
		Path: "Сценарий. Ютубер.md",
		Content: []byte(`---
free: true
---
Page with dot in name`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	// Main page should have embedded content
	mainPage := pages.PathMap["main.md"]
	require.NotNil(t, mainPage, "main.md should exist")
	require.NotEmpty(t, mainPage.HTML, "main.md should have non-empty HTML")

	// The link should NOT be marked as wip
	require.NotContains(t, string(mainPage.HTML), `class="wip"`,
		"Link to page with dot in name should not be marked as wip")

	// The link should have data-pid (proving page was found)
	require.Contains(t, string(mainPage.HTML), `data-pid=`,
		"Link should have data-pid since target page exists")
}

// TestMarkdownImageAssetReplace tests that standard markdown images ![alt](src)
// get their src replaced with the URL from AssetReplaces.
// Bug: Wikilinks ![[image.png]] work, but standard markdown images don't get replaced.
func TestMarkdownImageAssetReplace(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "note.md",
		Content: []byte(`![1499_0.jpg](./assets/1499_0.jpg)`),
		Assets: map[string]*model.NoteAssetReplace{
			"./assets/1499_0.jpg": {
				ID:           322,
				URL:          "http://example.com/replaced-url.jpg",
				Hash:         "abc123",
				AbsolutePath: "vault/assets/1499_0.jpg",
			},
		},
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/note"].HTML)

	// The image src should be replaced with the URL from AssetReplaces
	require.Contains(t, html, `src="http://example.com/replaced-url.jpg"`,
		"Markdown image src should be replaced with AssetReplaces URL")

	// Should NOT contain the original relative path
	require.NotContains(t, html, `src="./assets/1499_0.jpg"`,
		"Original relative path should be replaced")
}

// TestImageWithSameNameAsNote tests that image embeds ![[note.png]] are NOT
// resolved as note links when a note with the same basename exists.
// Bug: extractInLinks resolved ![[software.png]] as /software (the note)
// instead of keeping it as an image reference, breaking the page render.
func TestImageWithSameNameAsNote(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "software.md",
		Content: []byte(`---
free: true
---
![[software.png]]

Some content about software.`),
	}, {
		Path: "other.md",
		Content: []byte(`---
free: true
---
Link to [[software]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
		Version: "latest",
	})
	require.NoError(t, err)

	// software.md must have non-empty HTML
	softwarePage := pages.PathMap["software.md"]
	require.NotNil(t, softwarePage, "software.md should exist")
	require.NotEmpty(t, softwarePage.HTML, "software.md should have non-empty HTML")

	// The image should be rendered as <img>, not as a broken embed
	require.Contains(t, string(softwarePage.HTML), `<img src="software.png">`,
		"Image embed should be rendered as img tag")

	// Should NOT contain self-reference error or embed error
	require.NotContains(t, string(softwarePage.HTML), `/software?version`,
		"Image should not be resolved as a versioned link to the note")

	// other.md should link to software correctly
	otherPage := pages.PathMap["other.md"]
	require.NotNil(t, otherPage, "other.md should exist")
	require.Contains(t, string(otherPage.HTML), `href="/software"`,
		"Link to software should work correctly")
}

func TestNoteCacheReusesAST(t *testing.T) {
	log := logger.TestLogger{}

	// First load - no cache
	sources1 := []mdloader.SourceFile{
		{Path: "note1.md", Content: []byte("# Note 1\nContent 1")},
		{Path: "note2.md", Content: []byte("# Note 2\nContent 2")},
	}

	pages1, err := mdloader.Load(mdloader.Options{
		Sources: sources1,
		Log:     &log,
	})
	require.NoError(t, err)

	ast1Note1 := pages1.PathMap["note1.md"].Ast()
	ast1Note2 := pages1.PathMap["note2.md"].Ast()
	require.NotNil(t, ast1Note1)
	require.NotNil(t, ast1Note2)

	// Second load - note1 unchanged, note2 changed
	sources2 := []mdloader.SourceFile{
		{Path: "note1.md", Content: []byte("# Note 1\nContent 1")},       // same
		{Path: "note2.md", Content: []byte("# Note 2\nContent CHANGED")}, // changed
	}

	pages2, err := mdloader.Load(mdloader.Options{
		Sources: sources2,
		Log:     &log,
		NoteCache: func(source mdloader.SourceFile) *model.NoteView {
			old, ok := pages1.PathMap[source.Path]
			if !ok {
				return nil
			}
			if string(old.Content) == string(source.Content) {
				return old
			}
			return nil
		},
	})
	require.NoError(t, err)

	ast2Note1 := pages2.PathMap["note1.md"].Ast()
	ast2Note2 := pages2.PathMap["note2.md"].Ast()

	// note1 AST should be reused (same pointer)
	require.Same(t, ast1Note1, ast2Note1, "unchanged note should reuse AST from cache")

	// note2 AST should be different (re-parsed)
	require.NotSame(t, ast1Note2, ast2Note2, "changed note should have new AST")
}

// TestSlugMutualLinks tests that two notes with custom slugs can link to each
// other correctly. Links should resolve to the slug URL, not the original filename.
// Bug: When note_a (slug: /new/address) links to note_b and note_b links back,
// the links should use slug URLs, not the original filenames.
func TestSlugMutualLinks(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "note_a.md",
		Content: []byte(`---
free: true
slug: /new/address
title: Note A
---
Link to B: [[note_b]]`),
	}, {
		Path: "note_b.md",
		Content: []byte(`---
free: true
slug: /other/place
title: Note B
---
Link to A: [[note_a]]`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	// Note A should be accessible by its slug
	noteA := pages.Map["/new/address"]
	require.NotNil(t, noteA, "Note A should be accessible by slug /new/address")
	require.Equal(t, "/new/address", noteA.Permalink, "Note A permalink should be the slug")

	// Note B should be accessible by its slug
	noteB := pages.Map["/other/place"]
	require.NotNil(t, noteB, "Note B should be accessible by slug /other/place")
	require.Equal(t, "/other/place", noteB.Permalink, "Note B permalink should be the slug")

	// Note A's HTML should link to Note B's slug URL
	htmlA := string(noteA.HTML)
	require.Contains(t, htmlA, `href="/other/place"`,
		"Note A should link to Note B's slug URL, not original filename")
	require.NotContains(t, htmlA, `href="/note_b"`,
		"Note A should NOT link to Note B's original filename")

	// Note B's HTML should link to Note A's slug URL
	htmlB := string(noteB.HTML)
	require.Contains(t, htmlB, `href="/new/address"`,
		"Note B should link to Note A's slug URL, not original filename")
	require.NotContains(t, htmlB, `href="/note_a"`,
		"Note B should NOT link to Note A's original filename")

	// InLinks should use slug URLs, not original filenames
	require.Equal(t, map[string]struct{}{"/other/place": {}}, noteA.InLinks,
		"Note A's InLinks should contain Note B's slug URL")
	require.Equal(t, map[string]struct{}{"/new/address": {}}, noteB.InLinks,
		"Note B's InLinks should contain Note A's slug URL")

	// Neither note should have broken link warnings
	require.Empty(t, noteA.Warnings, "Note A should not have any warnings")
	require.Empty(t, noteB.Warnings, "Note B should not have any warnings")
}

// TestYouTubeEmbed tests that YouTube links in image syntax are rendered as embeds.
// Markdown: ![](https://www.youtube.com/watch?v=VIDEO_ID)
// Should render as YouTube embed iframe, not as broken image.
func TestYouTubeEmbed(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "video.md",
		Content: []byte(`---
free: true
---
Check out this video:

![](https://www.youtube.com/watch?v=SJCGVbYN9XY)

More content below.`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/video"].HTML)

	// Should contain YouTube embed wrapper
	require.Contains(t, html, `enclave-object-wrapper`,
		"YouTube link should be rendered as enclave embed")

	// Should contain YouTube-specific class
	require.Contains(t, html, `youtube-enclave-object`,
		"Should have youtube-enclave-object class")

	// Should contain the video ID in iframe src
	require.Contains(t, html, `youtube.com/embed/SJCGVbYN9XY`,
		"Should contain YouTube embed iframe with video ID")

	// Should NOT be rendered as regular image
	require.NotContains(t, html, `<img src="https://www.youtube.com`,
		"YouTube link should NOT be rendered as regular img tag")
}

// TestYouTubeShortLinkEmbed tests that youtu.be short links are also rendered as embeds.
func TestYouTubeShortLinkEmbed(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path: "video_short.md",
		Content: []byte(`---
free: true
---
![](https://youtu.be/SJCGVbYN9XY)`),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	html := string(pages.Map["/video_short"].HTML)

	// Should contain YouTube embed
	require.Contains(t, html, `youtube-enclave-object`,
		"youtu.be link should be rendered as YouTube embed")

	require.Contains(t, html, `youtube.com/embed/SJCGVbYN9XY`,
		"Should contain YouTube embed iframe with video ID")
}

// TestInLinksWithCachedAST tests that InLinks are correctly populated when
// notes are loaded with cached AST. This reproduces a bug where AST mutation
// in extractInLinks (changing link.Target from "note" to "/note") breaks
// subsequent reloads because the cached AST has already-resolved targets.
//
// Bug scenario:
// 1. First load: note_a and note_b exist, note_b links to note_a via [[note_a]]
// 2. extractInLinks resolves [[note_a]] and mutates AST: link.Target = "/note_a"
// 3. Second load with cache: note_c is added (also links to note_a)
// 4. For note_b, cached AST is reused with link.Target = "/note_a"
// 5. extractInLinks sees "/note_a", checks isSimpleFilename (has "/") = false
// 6. Falls through to path-based resolution, fails to find match
// 7. Result: note_a.InLinks is missing note_b.
func TestInLinksWithCachedAST(t *testing.T) {
	log := logger.TestLogger{}

	// First load - note_a and note_b, where note_b links to note_a
	sources1 := []mdloader.SourceFile{
		{Path: "note_a.md", Content: []byte("# Note A\nThis is note A")},
		{Path: "note_b.md", Content: []byte("# Note B\nLink to [[note_a]]")},
	}

	pages1, err := mdloader.Load(mdloader.Options{
		Sources: sources1,
		Log:     &log,
	})
	require.NoError(t, err)

	// Verify InLinks work correctly after first load
	require.Equal(t, map[string]struct{}{"/note_b": {}}, pages1.Map["/note_a"].InLinks,
		"note_a should have InLink from note_b after first load")

	// Second load with cache - add note_c that also links to note_a
	// note_a and note_b are unchanged (will use cached AST)
	sources2 := []mdloader.SourceFile{
		{Path: "note_a.md", Content: []byte("# Note A\nThis is note A")},             // unchanged
		{Path: "note_b.md", Content: []byte("# Note B\nLink to [[note_a]]")},         // unchanged
		{Path: "note_c.md", Content: []byte("# Note C\nAnother link to [[note_a]]")}, // new
	}

	pages2, err := mdloader.Load(mdloader.Options{
		Sources: sources2,
		Log:     &log,
		NoteCache: func(source mdloader.SourceFile) *model.NoteView {
			old, ok := pages1.PathMap[source.Path]
			if !ok {
				return nil
			}
			if string(old.Content) == string(source.Content) {
				return old // Return cached note (with mutated AST)
			}
			return nil
		},
	})
	require.NoError(t, err)

	// After second load, note_a should have InLinks from BOTH note_b and note_c
	expectedInLinks := map[string]struct{}{
		"/note_b": {},
		"/note_c": {},
	}
	require.Equal(t, expectedInLinks, pages2.Map["/note_a"].InLinks,
		"note_a should have InLinks from both note_b and note_c after cached reload")
}

// TestPathBasedPatchApplied verifies that a patch using std.startsWith(path, ...)
// correctly adds fields to the note's RawMeta.
func TestPathBasedPatchApplied(t *testing.T) {
	log := logger.TestLogger{}

	patch := frontmatterpatch.Compile(
		1,
		[]string{"patch_tests/*"}, // includePatterns
		nil,                        // excludePatterns
		`if std.startsWith(path, "patch_tests/") then { patch_applied: true } else {}`,
		0,
		"path-based logic",
	)

	sources := []mdloader.SourceFile{{
		Path: "patch_tests/path_based.md",
		Content: []byte("---\nfree: true\nlayout: meta_inspector\ntitle: Path-Based Logic Test\n---\nContent"),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources:            sources,
		Log:                &log,
		FrontmatterPatches: []frontmatterpatch.CompiledPatch{patch},
	})
	require.NoError(t, err)

	note := pages.PathMap["patch_tests/path_based.md"]
	require.NotNil(t, note, "note should exist")

	// patch_applied should be true (added by path-based patch)
	require.Equal(t, true, note.RawMeta["patch_applied"],
		"path-based patch should add patch_applied: true to RawMeta")

	// free should still be true (from frontmatter)
	require.Equal(t, true, note.RawMeta["free"],
		"free field should remain true from frontmatter")

	// patch should be recorded as applied
	require.Len(t, note.AppliedFrontmatterPatches, 1,
		"one patch should be recorded as applied")
}

// TestPathBasedPatchNotAppliedToNonMatchingPath verifies that a path-based patch
// does NOT apply to notes outside the matching path prefix.
func TestPathBasedPatchNotAppliedToNonMatchingPath(t *testing.T) {
	log := logger.TestLogger{}

	patch := frontmatterpatch.Compile(
		1,
		[]string{"patch_tests/*"},
		nil,
		`if std.startsWith(path, "patch_tests/") then { patch_applied: true } else {}`,
		0,
		"path-based logic",
	)

	sources := []mdloader.SourceFile{{
		Path:    "other_folder/note.md",
		Content: []byte("---\nfree: true\n---\nContent"),
	}}

	pages, err := mdloader.Load(mdloader.Options{
		Sources:            sources,
		Log:                &log,
		FrontmatterPatches: []frontmatterpatch.CompiledPatch{patch},
	})
	require.NoError(t, err)

	note := pages.PathMap["other_folder/note.md"]
	require.NotNil(t, note, "note should exist")

	// patch_applied should NOT be present - patch only applies to patch_tests/*
	_, hasPatchApplied := note.RawMeta["patch_applied"]
	require.False(t, hasPatchApplied,
		"patch_applied should NOT be in RawMeta for non-matching path")

	// No patches should be applied
	require.Empty(t, note.AppliedFrontmatterPatches,
		"no patches should be applied for non-matching path")
}

// TestFrontmatterPatchNotDoubledOnCacheHit verifies that a frontmatter patch
// (e.g. title suffix) is applied exactly once even when NoteCache returns a
// cached NoteView on subsequent loads.
//
// Regression: cached.RawMeta already contained the patched title, so the
// patch was applied again on every reload — accumulating "— Site" N times.
func TestFrontmatterPatchNotDoubledOnCacheHit(t *testing.T) {
	log := logger.TestLogger{}

	noteContent := []byte("---\ntitle: Original Title\nfree: true\n---\nContent")

	source := mdloader.SourceFile{
		Path:    "post.md",
		Content: noteContent,
	}

	patch := frontmatterpatch.Compile(1, []string{"*"}, nil, `meta + { title: meta.title + " — Site" }`, 0, "title suffix")

	makeNoteCache := func(prev *model.NoteViews) func(mdloader.SourceFile) *model.NoteView {
		return func(src mdloader.SourceFile) *model.NoteView {
			if prev == nil {
				return nil
			}
			old, ok := prev.PathMap[src.Path]
			if !ok {
				return nil
			}
			if string(old.Content) == string(src.Content) {
				return old
			}
			return nil
		}
	}

	// First load — no cache
	pages1, err := mdloader.Load(mdloader.Options{
		Sources:            []mdloader.SourceFile{source},
		Log:                &log,
		FrontmatterPatches: []frontmatterpatch.CompiledPatch{patch},
	})
	require.NoError(t, err)
	require.Equal(t, "Original Title — Site", pages1.PathMap["post.md"].Title,
		"patch should be applied once on first load")

	// Second load — same content, NoteCache will hit
	pages2, err := mdloader.Load(mdloader.Options{
		Sources:            []mdloader.SourceFile{source},
		Log:                &log,
		FrontmatterPatches: []frontmatterpatch.CompiledPatch{patch},
		NoteCache:          makeNoteCache(pages1),
	})
	require.NoError(t, err)
	require.Equal(t, "Original Title — Site", pages2.PathMap["post.md"].Title,
		"patch must NOT be applied twice on cached reload")

	// Third load — same content, NoteCache will hit again
	pages3, err := mdloader.Load(mdloader.Options{
		Sources:            []mdloader.SourceFile{source},
		Log:                &log,
		FrontmatterPatches: []frontmatterpatch.CompiledPatch{patch},
		NoteCache:          makeNoteCache(pages2),
	})
	require.NoError(t, err)
	require.Equal(t, "Original Title — Site", pages3.PathMap["post.md"].Title,
		"patch must NOT accumulate on third cached reload")
}
