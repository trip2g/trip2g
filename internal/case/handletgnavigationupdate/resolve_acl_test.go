package handletgnavigationupdate

import (
	"context"
	"database/sql"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestResolveOpenUsesLiveNotesOnly(t *testing.T) {
	liveViews := noteViewsWith(&model.NoteView{
		PathID:  7,
		Path:    "public.md",
		Title:   "Public",
		Content: []byte("public body"),
		Free:    true,
	})

	sentTexts := make([]string, 0, 2)
	env := newNavigationTestEnv(t, liveViews, &sentTexts)

	err := ResolveOpen(context.Background(), env, privateStartUpdate("/start browse_99"), 99)
	require.NoError(t, err)
	require.Len(t, sentTexts, 1)
	require.Equal(t, "Note not found (99).", sentTexts[0])
	require.NotContains(t, sentTexts[0], "private body")
}

func TestResolveOpenDoesNotRenderPaidLiveNote(t *testing.T) {
	liveViews := noteViewsWith(&model.NoteView{
		PathID:  11,
		Path:    "paid.md",
		Title:   "Paid",
		Content: []byte("paid body"),
		Free:    false,
	})

	sentTexts := make([]string, 0, 2)
	env := newNavigationTestEnv(t, liveViews, &sentTexts)

	err := ResolveOpen(context.Background(), env, privateStartUpdate("/start browse_11"), 11)
	require.NoError(t, err)
	require.Len(t, sentTexts, 1)
	require.Equal(t, "Note not found (11).", sentTexts[0])
	require.NotContains(t, sentTexts[0], "paid body")
}

func TestResolveOpenRendersLiveNote(t *testing.T) {
	liveViews := noteViewsWith(&model.NoteView{
		PathID:  7,
		Path:    "public.md",
		Title:   "Public",
		Content: []byte("public body"),
		Free:    true,
	})

	sentTexts := make([]string, 0, 2)
	env := newNavigationTestEnv(t, liveViews, &sentTexts)

	err := ResolveOpen(context.Background(), env, privateStartUpdate("/start browse_7"), 7)
	require.NoError(t, err)
	require.Len(t, sentTexts, 1)
	require.Contains(t, sentTexts[0], "public body")
}

func noteViewsWith(notes ...*model.NoteView) *model.NoteViews {
	nvs := model.NewNoteViews()
	nvs.BasenameMap = make(map[string][]*model.NoteView)
	md := goldmark.New()
	for _, note := range notes {
		note.SetAst(md.Parser().Parse(text.NewReader(note.Content)))
		nvs.List = append(nvs.List, note)
		nvs.PathMap[note.Path] = note
	}
	return nvs
}

func newNavigationTestEnv(t *testing.T, liveViews *model.NoteViews, sentTexts *[]string) *EnvMock {
	t.Helper()
	return &EnvMock{
		BotIDFunc:         func() int64 { return 1 },
		BotLinkFunc:       func() string { return "https://t.me/test_bot" },
		LiveNoteViewsFunc: func() *model.NoteViews { return liveViews },
		LoggerFunc:        func() logger.Logger { return &logger.DummyLogger{} },
		TgUserStateByBotIDAndChatIDFunc: func(context.Context, db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
			return db.TgUserState{}, sql.ErrNoRows
		},
		UpsertTgUserStateFunc: func(context.Context, db.UpsertTgUserStateParams) error { return nil },
		SendFunc: func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
			switch cfg := msg.(type) {
			case tgbotapi.MessageConfig:
				*sentTexts = append(*sentTexts, cfg.Text)
			case tgbotapi.EditMessageTextConfig:
				*sentTexts = append(*sentTexts, cfg.Text)
			default:
				t.Fatalf("unexpected message type %T", msg)
			}
			return tgbotapi.Message{MessageID: len(*sentTexts)}, nil
		},
	}
}

func privateStartUpdate(text string) tgbotapi.Update {
	return tgbotapi.Update{Message: &tgbotapi.Message{
		MessageID: 1,
		Text:      text,
		Chat:      &tgbotapi.Chat{ID: 123, Type: "private"},
		Entities:  []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 6}},
	}}
}
