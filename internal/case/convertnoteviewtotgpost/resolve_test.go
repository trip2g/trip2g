package convertnoteviewtotgpost_test

import (
	"context"
	"os"
	"testing"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func TestContent(t *testing.T) {
	var env struct{}

	obsidianMarkdown, err := os.ReadFile("obsidian.md")
	require.NoError(t, err)

	tgMarkdown, err := os.ReadFile("telegram.md")
	require.NoError(t, err)

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Content: []byte(`---
free: true
title: "Sample Note"
---
` + string(obsidianMarkdown)),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), &env, nvs.List[0])
	require.NoError(t, err)

	require.Empty(t, post.Warnings)
	require.Equal(t, string(tgMarkdown), post.Content)
}
