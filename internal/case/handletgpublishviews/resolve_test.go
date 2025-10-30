package handletgpublishviews_test

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/case/handletgpublishviews"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"math/rand/v2"

	"github.com/stretchr/testify/require"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg handletgpublishviews_test . Env

func testLogger() logger.Logger {
	return &logger.TestLogger{}
}

func insertTelegramPublishTags(_ context.Context, _ string) error {
	return nil
}

func makeTelegramPublishTagByLabel() func(context.Context, string) (db.TelegramPublishTag, error) {
	r := rand.New(rand.NewPCG(0, 0))

	return func(_ context.Context, label string) (db.TelegramPublishTag, error) {
		return db.TelegramPublishTag{
			ID:    int64(r.Uint64()),
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

var timeLocation = time.FixedZone("testzone", 7*3600) //nolint:gochecknoglobals // it's ok for tests

func prepare(t *testing.T, nvs *model.NoteViews) *EnvMock {
	env := &EnvMock{
		LoggerFunc: testLogger,

		TimeLocationFunc: func() *time.Location {
			return timeLocation
		},

		LatestNoteViewsFunc: func() *model.NoteViews {
			return nvs
		},

		InsertTelegramPublishTagsFunc: insertTelegramPublishTags,
		TelegramPublishTagByLabelFunc: makeTelegramPublishTagByLabel(),

		UpsertTelegramPublishNoteFunc:    upsertTelegramPublishNote,
		UpsertTelegramPublishNoteTagFunc: upsertTelegramPublishNoteTag,

		DeleteTelegramPublishNoteTagsByPathIDFunc: deleteTelegramPublishNoteTagsByPathID,

		SendTelegramPublishPostFunc: func(ctx context.Context, pathID int64, instant bool) error {
			return nil
		},

		UpdateTelegramPublishPostFunc: func(ctx context.Context, pathID int64) error {
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	changedPathIDs := []int64{}

	for _, nv := range nvs.List {
		changedPathIDs = append(changedPathIDs, nv.PathID)
	}

	err := handletgpublishviews.Resolve(ctx, env, changedPathIDs)
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

	require.Empty(t, nvs.List[0].Warnings)
	require.Empty(t, nvs.List[1].Warnings)
	require.Len(t, nvs.List[2].Warnings, 1)
	require.Len(t, nvs.List[3].Warnings, 1)
	require.Empty(t, nvs.List[4].Warnings)

	require.Len(t, env.calls.InsertTelegramPublishTags, 2)
	require.Equal(t, "tag1", env.calls.InsertTelegramPublishTags[0].Label)
	require.Equal(t, "tag2", env.calls.InsertTelegramPublishTags[1].Label)

	require.Len(t, env.calls.UpsertTelegramPublishNote, 2)
	require.Equal(t, int64(7), env.calls.UpsertTelegramPublishNote[0].Params.NotePathID)
	require.Equal(t, int64(9), env.calls.UpsertTelegramPublishNote[1].Params.NotePathID)

	// should be converted to UTC
	require.Equal(t, time.Date(2024, 7, 2, 16, 2, 0, 0, time.UTC), env.calls.UpsertTelegramPublishNote[0].Params.PublishAt)
	require.Equal(t, time.Date(2024, 7, 2, 20, 2, 0, 0, time.UTC), env.calls.UpsertTelegramPublishNote[1].Params.PublishAt)

	require.Len(t, env.calls.SendTelegramPublishPost, 2)
	require.Equal(t, int64(7), env.calls.SendTelegramPublishPost[0].NotePathID)
	require.Equal(t, int64(9), env.calls.SendTelegramPublishPost[1].NotePathID)
	require.True(t, env.calls.SendTelegramPublishPost[0].Instant)
	require.True(t, env.calls.SendTelegramPublishPost[0].Instant)
}
