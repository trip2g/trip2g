package mdloader

import (
	"testing"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
)

func TestFlatIndexFirstSecond(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []SourceFile{{
		Path:    "index.md",
		Content: []byte(`Hello [[first]] [[second]]`),
	}, {
		Path: "first.md",
		Content: []byte(`---
title: First
---

First. Second [[second]]`),
	}, {
		Path:    "second.md",
		Content: []byte(`Second.`),
	}}

	pages, err := Load(sourceFiles, &log)
	require.NoError(t, err)
	require.Len(t, pages, 3)

	require.Equal(t, "index", pages["/index"].Title)
	require.Equal(t, "First", pages["/first"].Title)
	require.Equal(t, "second", pages["/second"].Title)

	require.Equal(t, map[string]struct{}{}, pages["/index"].InLinks)
	require.Equal(t, map[string]struct{}{"/index": {}}, pages["/first"].InLinks)
	require.Equal(t, map[string]struct{}{"/index": {}, "/first": {}}, pages["/second"].InLinks)
}
