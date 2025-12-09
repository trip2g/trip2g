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
	now       time.Time
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

func (e *testEnv) ListTelegramPublishSentAccountMessagesByAccountAndChat(
	ctx context.Context,
	arg db.ListTelegramPublishSentAccountMessagesByAccountAndChatParams,
) ([]db.ListTelegramPublishSentAccountMessagesByAccountAndChatRow, error) {
	return nil, nil
}

func (e *testEnv) PublicURL() string {
	return e.publicURL
}

func (e *testEnv) TimeLocation() *time.Location {
	return time.UTC
}

func (e *testEnv) Now() time.Time {
	// Default to a fixed time if not set
	if e.now.IsZero() {
		return time.Date(2025, 11, 5, 14, 0, 0, 0, time.UTC)
	}
	return e.now
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
	// Current time: 2025-11-05 14:00:00
	now := time.Date(2025, 11, 5, 14, 0, 0, 0, time.UTC)
	// Publish time: 2025-11-05 15:15:00 (75 minutes away - more than 30 minutes)
	publishAt := now.Add(75 * time.Minute)

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
telegram_publish_at: "` + publishAt.Format("2006-01-02T15:04:05") + `"
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
		now:       now,
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

	// Check date formatting (15:15 because publish time is 75 minutes away from 14:00)
	require.Contains(t, post.Content, "5 ноября, 15:15")

	// Check footer structure
	require.Contains(t, post.Content, "🔜 Скоро выйдут:")
	require.Contains(t, post.Content, "📬 Подпишитесь, чтобы не пропустить")
}

func TestChatIDZeroSkipsDBQuery(t *testing.T) {
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

	// Create a custom env that tracks if ListTelegramPublishSentMessagesByChatID was called
	called := false
	env := &testEnvWithTracking{
		testEnv: testEnv{
			nvs:       nvs,
			logger:    &logger.TestLogger{},
			sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
			publicURL: "https://example.com",
		},
		onListCalled: func() {
			called = true
		},
	}

	source := model.TelegramPostSource{
		NoteView: nvs.List[0],
		ChatID:   0, // ChatID is 0
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	require.Empty(t, post.Warnings)
	require.Equal(t, "hello", post.Content)

	// Verify that ListTelegramPublishSentMessagesByChatID was NOT called
	require.False(t, called, "ListTelegramPublishSentMessagesByChatID should not be called when ChatID is 0")
}

func TestUnpublishedLinkMoreThan30MinutesAway(t *testing.T) {
	// Current time: 2025-11-05 14:00:00
	now := time.Date(2025, 11, 5, 14, 0, 0, 0, time.UTC)
	// Publish time: 2025-11-05 15:00:00 (60 minutes away)
	publishAt := now.Add(60 * time.Minute)

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path: "main.md",
				Content: []byte(`---
free: true
title: "Main Note"
---
Link to future post: [[future_post]].`),
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
				Path: "future_post.md",
				Content: []byte(`---
free: true
title: "Future Post"
telegram_publish_at: "` + publishAt.Format("2006-01-02T15:04:05") + `"
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
	unpubNote.PathID = 999
	nvs.Map["future_post"] = unpubNote

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
		now:       now,
	}

	mainNote := nvs.List[0]
	source := model.TelegramPostSource{
		NoteView: mainNote,
		ChatID:   123,
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	// Should have underlined text in content
	require.Contains(t, post.Content, "<u>Future Post</u>")

	// Should have footer with publish date (more than 30 minutes away)
	require.Contains(t, post.Content, "🔜 Скоро выйдут:")
	require.Contains(t, post.Content, "Future Post")
	require.Contains(t, post.Content, "5 ноября, 15:00")
}

func TestUnpublishedLinkExactly30MinutesAway(t *testing.T) {
	// Current time: 2025-11-05 14:00:00
	now := time.Date(2025, 11, 5, 14, 0, 0, 0, time.UTC)
	// Publish time: 2025-11-05 14:30:00 (exactly 30 minutes away)
	publishAt := now.Add(30 * time.Minute)

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path: "main.md",
				Content: []byte(`---
free: true
title: "Main Note"
---
Link to soon post: [[soon_post]].`),
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
				Path: "soon_post.md",
				Content: []byte(`---
free: true
title: "Soon Post"
telegram_publish_at: "` + publishAt.Format("2006-01-02T15:04:05") + `"
telegram_publish_tags: ["tag1"]
---
Soon content`),
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
	unpubNote.PathID = 999
	nvs.Map["soon_post"] = unpubNote

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
		now:       now,
	}

	mainNote := nvs.List[0]
	source := model.TelegramPostSource{
		NoteView: mainNote,
		ChatID:   123,
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	// Should have underlined text in content
	require.Contains(t, post.Content, "<u>Soon Post</u>")

	// Should NOT have footer (within 30 minutes)
	require.NotContains(t, post.Content, "🔜 Скоро выйдут:")
	require.NotContains(t, post.Content, "14:30")
}

func TestUnpublishedLinkLessThan30MinutesAway(t *testing.T) {
	// Current time: 2025-11-05 14:00:00
	now := time.Date(2025, 11, 5, 14, 0, 0, 0, time.UTC)
	// Publish time: 2025-11-05 14:15:00 (15 minutes away)
	publishAt := now.Add(15 * time.Minute)

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path: "main.md",
				Content: []byte(`---
free: true
title: "Main Note"
---
Link to very soon post: [[very_soon_post]].`),
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
				Path: "very_soon_post.md",
				Content: []byte(`---
free: true
title: "Very Soon Post"
telegram_publish_at: "` + publishAt.Format("2006-01-02T15:04:05") + `"
telegram_publish_tags: ["tag1"]
---
Very soon content`),
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
	unpubNote.PathID = 999
	nvs.Map["very_soon_post"] = unpubNote

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
		now:       now,
	}

	mainNote := nvs.List[0]
	source := model.TelegramPostSource{
		NoteView: mainNote,
		ChatID:   123,
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	// Should have underlined text in content
	require.Contains(t, post.Content, "<u>Very Soon Post</u>")

	// Should NOT have footer (within 30 minutes)
	require.NotContains(t, post.Content, "🔜 Скоро выйдут:")
	require.NotContains(t, post.Content, "14:15")
}

func TestUnpublishedLinkInThePast(t *testing.T) {
	// Current time: 2025-11-05 14:00:00
	now := time.Date(2025, 11, 5, 14, 0, 0, 0, time.UTC)
	// Publish time: 2025-11-05 13:00:00 (1 hour in the past)
	publishAt := now.Add(-60 * time.Minute)

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path: "main.md",
				Content: []byte(`---
free: true
title: "Main Note"
---
Link to past post: [[past_post]].`),
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
				Path: "past_post.md",
				Content: []byte(`---
free: true
title: "Past Post"
telegram_publish_at: "` + publishAt.Format("2006-01-02T15:04:05") + `"
telegram_publish_tags: ["tag1"]
---
Past content`),
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
	unpubNote.PathID = 999
	nvs.Map["past_post"] = unpubNote

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
		now:       now,
	}

	mainNote := nvs.List[0]
	source := model.TelegramPostSource{
		NoteView: mainNote,
		ChatID:   123,
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	// Should have underlined text in content
	require.Contains(t, post.Content, "<u>Past Post</u>")

	// Should NOT have footer (publish time in the past)
	require.NotContains(t, post.Content, "🔜 Скоро выйдут:")
	require.NotContains(t, post.Content, "13:00")
}

func TestMissingAssets(t *testing.T) {
	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path: "test.md",
			Content: []byte(`---
free: true
title: "Note with Image"
---
hello

![image](test.jpg)`),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)
	require.Len(t, nvs.List, 1)

	// Add asset reference without AssetReplace (simulating asset not yet uploaded)
	note := nvs.List[0]
	note.Assets = map[string]struct{}{
		"test.jpg": {},
	}
	note.AssetReplaces = map[string]*model.NoteAssetReplace{} // Empty - no uploaded assets

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
	}

	source := model.TelegramPostSource{
		NoteView: note,
		ChatID:   123,
		Instant:  false,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.Error(t, err)
	require.Nil(t, post)

	// Verify it's the correct error type
	var assetsErr *convertnoteviewtotgpost.ErrAssetsNotReadyError
	require.ErrorAs(t, err, &assetsErr)
	require.Contains(t, assetsErr.MissingAssets, "test.jpg")
}

type testEnvWithTracking struct {
	testEnv
	onListCalled func()
}

func (e *testEnvWithTracking) ListTelegramPublishSentMessagesByChatID(
	ctx context.Context,
	chatID int64,
) ([]db.ListTelegramPublishSentMessagesByChatIDRow, error) {
	if e.onListCalled != nil {
		e.onListCalled()
	}
	return e.testEnv.ListTelegramPublishSentMessagesByChatID(ctx, chatID)
}
