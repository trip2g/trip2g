package convertnoteviewtotgpost_test

import (
	"context"
	"testing"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type testEnv struct {
	nvs       *model.NoteViews
	logger    logger.Logger
	sentMsgs  []db.ListAllTelegramPublishSentMessagesRow
	publicURL string
}

func (e *testEnv) LatestNoteViews() *model.NoteViews {
	return e.nvs
}

func (e *testEnv) Logger() logger.Logger {
	return e.logger
}

func (e *testEnv) ListAllTelegramPublishSentMessages(ctx context.Context) ([]db.ListAllTelegramPublishSentMessagesRow, error) {
	return e.sentMsgs, nil
}

func (e *testEnv) PublicURL() string {
	return e.publicURL
}

func TestContent(t *testing.T) {
	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Content: []byte(`---
free: true
title: "Sample Note"
---
hello`),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListAllTelegramPublishSentMessagesRow{},
		publicURL: "https://example.com",
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, nvs.List[0])
	require.NoError(t, err)

	require.Empty(t, post.Warnings)
	require.Equal(t, "hello", post.Content)
}
