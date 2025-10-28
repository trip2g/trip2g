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
	sentMsgs  []db.ListTelegramPublishSentMessagesByChatIDRow
	publicURL string
}

func (e *testEnv) LatestNoteViews() *model.NoteViews {
	return e.nvs
}

func (e *testEnv) Logger() logger.Logger {
	return e.logger
}

func (e *testEnv) ListTelegramPublishSentMessagesByChatID(ctx context.Context, chatID int64) ([]db.ListTelegramPublishSentMessagesByChatIDRow, error) {
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
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, nvs.List[0], 123)
	require.NoError(t, err)

	require.Empty(t, post.Warnings)
	require.Equal(t, "hello", post.Content)
}
