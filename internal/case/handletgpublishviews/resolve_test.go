package handletgpublishviews_test

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/case/handletgpublishviews"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg handletgpublishviews_test . Env

func testLogger() logger.Logger {
	return &logger.TestLogger{}
}

func insertTelegramPublishTags(_ context.Context, _ string) error {
	return nil
}

func makeTelegramPublishTagByLabel() func(context.Context, string) (db.TelegramPublishTag, error) {
	r := rand.New(rand.NewSource(0))

	return func(_ context.Context, label string) (db.TelegramPublishTag, error) {
		return db.TelegramPublishTag{
			ID:    r.Int63(),
			Label: label,
		}, nil
	}
}

func upsertTelegramPublishNote(ctx context.Context, params db.UpsertTelegramPublishNoteParams) error {
	return nil
}

func deleteTelegramPublishNoteTagsByPathID(ctx context.Context, pathID int64) error {
	return nil
}

func upsertTelegramPublishNoteTag(ctx context.Context, params db.UpsertTelegramPublishNoteTagParams) error {
	return nil
}

var timeLocation = time.FixedZone("testzone", 7*3600)

func prepare(t *testing.T, nvs *model.NoteViews) *EnvMock {
	env := &EnvMock{
		LoggerFunc: testLogger,

		TimeLocationFunc: func() *time.Location {
			return timeLocation
		},

		InsertTelegramPublishTagsFunc: insertTelegramPublishTags,
		TelegramPublishTagByLabelFunc: makeTelegramPublishTagByLabel(),

		UpsertTelegramPublishNoteFunc:    upsertTelegramPublishNote,
		UpsertTelegramPublishNoteTagFunc: upsertTelegramPublishNoteTag,

		DeleteTelegramPublishNoteTagsByPathIDFunc: deleteTelegramPublishNoteTagsByPathID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := handletgpublishviews.Resolve(ctx, env, nvs)
	require.NoError(t, err)

	return env
}

func TestMetaExtractrsEmpty(t *testing.T) {
	nvs := model.NoteViews{}
	prepare(t, &nvs)
}

func TestMetaExtractrs(t *testing.T) {
	nvs := model.NoteViews{
		List: []*model.NoteView{{
			Path:    "1.md",
			RawMeta: map[string]any{},
		}, {
			Path:   "2.md",
			PathID: 7,
			RawMeta: map[string]any{
				"telegram_publish_at":   "2024-07-02T23:02:00",
				"telegram_publish_tags": []any{"tag1", "tag2"},
			},
		}, {
			Path: "3.md",
			RawMeta: map[string]any{
				"telegram_publish_at": "2024-07-02T23:02:00",
			},
		}, {
			Path: "4.md",
			RawMeta: map[string]any{
				"telegram_publish_tags": []any{"tag1"},
			},
		}, {
			Path:   "5.md",
			PathID: 9,
			RawMeta: map[string]any{
				"telegram_publish_at":   "2024-07-02T23:02:00+03:00", // with timezone
				"telegram_publish_tags": []any{"tag1"},
			},
		}},
	}

	env := prepare(t, &nvs)

	require.Len(t, nvs.List[0].Warnings, 0)
	require.Len(t, nvs.List[1].Warnings, 0)
	require.Len(t, nvs.List[2].Warnings, 1)
	require.Len(t, nvs.List[3].Warnings, 1)
	require.Len(t, nvs.List[4].Warnings, 0)

	require.Len(t, env.calls.InsertTelegramPublishTags, 2)
	require.Equal(t, "tag1", env.calls.InsertTelegramPublishTags[0].Label)
	require.Equal(t, "tag2", env.calls.InsertTelegramPublishTags[1].Label)

	require.Len(t, env.calls.UpsertTelegramPublishNote, 2)
	require.Equal(t, int64(7), env.calls.UpsertTelegramPublishNote[0].Params.NotePathID)
	require.Equal(t, int64(9), env.calls.UpsertTelegramPublishNote[1].Params.NotePathID)

	// should be converted to UTC
	require.Equal(t, time.Date(2024, 7, 2, 16, 2, 0, 0, time.UTC), env.calls.UpsertTelegramPublishNote[0].Params.PublishAt)
	require.Equal(t, time.Date(2024, 7, 2, 20, 2, 0, 0, time.UTC), env.calls.UpsertTelegramPublishNote[1].Params.PublishAt)
}
