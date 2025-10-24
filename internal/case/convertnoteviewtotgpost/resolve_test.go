package convertnoteviewtotgpost_test

import (
	"context"
	"testing"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func TestContent(t *testing.T) {
	var env struct{}

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Content: []byte(`---
free: true
title: "Sample Note"
---

hello
`),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), &env, nvs.List[0])
	require.NoError(t, err)

	require.Empty(t, post.Warnings)
	require.Equal(t, "hello\n", post.Content)
}
