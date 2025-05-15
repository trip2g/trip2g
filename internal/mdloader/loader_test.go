package mdloader_test

import (
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

	pages, err := mdloader.Load(sourceFiles, &log)
	require.NoError(t, err)
	require.Len(t, pages.Map, 3)

	require.Equal(t, "index", pages.Map["/index"].Title)
	require.Equal(t, "First", pages.Map["/first"].Title)
	require.Equal(t, "second", pages.Map["/second"].Title)

	require.Equal(t, map[string]struct{}{}, pages.Map["/index"].InLinks)
	require.Equal(t, map[string]struct{}{"/index": {}}, pages.Map["/first"].InLinks)
	require.Equal(t, map[string]struct{}{"/index": {}, "/first": {}}, pages.Map["/second"].InLinks)

	require.Equal(t, []string{"/dead"}, pages.Map["/first"].DeadLinks)
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

	pages, err := mdloader.Load(sourceFiles, &log)
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
Hello [[hidden]]`),
	}, {
		Path:    "hidden.md",
		Content: []byte(`Payed content`),
	}}

	pages, err := mdloader.Load(sourceFiles, &log)
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

	pages, err := mdloader.Load(sourceFiles, &log)
	require.NoError(t, err)

	require.Equal(t, pages.Map["/index"].Assets, map[string]struct{}{
		"image.png":  struct{}{},
		"/file.pdf":  struct{}{},
		"image2.png": struct{}{},
	})
}
