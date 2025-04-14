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
	require.Len(t, pages, 3)

	require.Equal(t, "index", pages["/index"].Title)
	require.Equal(t, "First", pages["/first"].Title)
	require.Equal(t, "second", pages["/second"].Title)

	require.Equal(t, map[string]struct{}{}, pages["/index"].InLinks)
	require.Equal(t, map[string]struct{}{"/index": {}}, pages["/first"].InLinks)
	require.Equal(t, map[string]struct{}{"/index": {}, "/first": {}}, pages["/second"].InLinks)

	require.Equal(t, []string{"/dead"}, pages["/first"].DeadLinks)
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
	require.Len(t, pages, 3)

	require.Equal(t, map[string]struct{}{}, pages["/second"].InLinks)
	require.Equal(t, map[string]struct{}{"/second": {}}, pages["/nested/first"].InLinks)
	require.Equal(t, map[string]struct{}{"/nested/first": {}}, pages["/nested/second"].InLinks)

	htmlSources := map[string]string{}

	for path, page := range pages {
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

	for path, page := range pages {
		htmlSources[path] = string(page.HTML)
	}

	cupaloy.SnapshotT(t, htmlSources)
}
