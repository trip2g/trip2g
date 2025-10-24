package markdownv2_test

import (
	"os"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/markdownv2"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func TestContent(t *testing.T) {
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

	convertor := markdownv2.CommonConverter{}

	res := convertor.Process(nvs.List[0])

	require.Empty(t, res.Warnings)
	require.Equal(t, string(tgMarkdown), res.Content)
}
