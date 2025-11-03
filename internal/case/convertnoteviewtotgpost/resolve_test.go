package convertnoteviewtotgpost_test

import (
	"context"
	"testing"
	"time"
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

func (e *testEnv) TimeLocation() *time.Location {
	return time.UTC
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

	source := model.TelegramPostSource{
		NoteView: nvs.List[0],
		ChatID:   123,
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	require.Empty(t, post.Warnings)
	require.Equal(t, "hello", post.Content)
}

func TestUnpublishedLinkUsesTitle(t *testing.T) {
	// Load main note
	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path: "main.md",
				Content: []byte(`---
free: true
title: "Main Note"
---
Специальное дополнение: [[second_reconsolidation3/metod_fejnmana_v_obuchenii_yazyikam]].`),
			},
		},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	// Load unpublished note separately
	mdOptionsUnpub := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path: "second_reconsolidation3/metod_fejnmana_v_obuchenii_yazyikam.md",
				Content: []byte(`---
free: true
title: "Метод Фейнмана в обучении языкам"
telegram_publish_at: "2025-11-05T14:15:00"
telegram_publish_tags: ["tag1"]
---
Future content`),
			},
		},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvsUnpub, err := mdloader.Load(mdOptionsUnpub)
	require.NoError(t, err)
	require.Len(t, nvsUnpub.List, 1)

	// Add unpublished note to the map
	unpubNote := nvsUnpub.List[0]
	unpubNote.PathID = 999 // Set a unique ID
	nvs.Map["second_reconsolidation3/metod_fejnmana_v_obuchenii_yazyikam"] = unpubNote

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
	}

	// Find main note
	var mainNote *model.NoteView
	for _, nv := range nvs.List {
		if nv.Path == "main.md" {
			mainNote = nv
			break
		}
	}
	require.NotNil(t, mainNote)

	source := model.TelegramPostSource{
		NoteView: mainNote,
		ChatID:   123,
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	// Check that content has underlined text
	require.Contains(t, post.Content, "<u>")

	// IMPORTANT: Check that footer uses title, not file path
	require.Contains(t, post.Content, "Метод Фейнмана в обучении языкам")
	require.NotContains(t, post.Content, "second_reconsolidation3/metod_fejnmana_v_obuchenii_yazyikam")

	// Check date formatting
	require.Contains(t, post.Content, "5 ноября, 14:15")

	// Check footer structure
	require.Contains(t, post.Content, "🔜 Скоро выйдут:")
	require.Contains(t, post.Content, "📬 Подпишитесь, чтобы не пропустить")
}
