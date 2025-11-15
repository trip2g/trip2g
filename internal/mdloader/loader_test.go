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

	require.Equal(t, map[string]struct{}{}, pages.Map["/second"].InLinks)
	require.Equal(t, map[string]struct{}{"/second": {}}, pages.Map["/nested/first"].InLinks)
	require.Equal(t, map[string]struct{}{"/nested/first": {}}, pages.Map["/nested/second"].InLinks)

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
