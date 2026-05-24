package handletgcanvasupdate

import (
	"context"
	"database/sql"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/obsidiancanvas"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

func newTestCanvas() []byte {
	return []byte(`{
		"nodes":[
			{"id":"start","type":"text","text":"START"},
			{"id":"intro","type":"text","text":"Welcome!"},
			{"id":"page","type":"text","text":"Page content"}
		],
		"edges":[
			{"id":"e1","fromNode":"start","toNode":"intro"},
			{"id":"e2","fromNode":"intro","toNode":"page","label":"Next"}
		]
	}`)
}

func newTestEnv(canvasPath string, canvasContent []byte) *EnvMock {
	nvs := model.NewNoteViews()
	if canvasContent != nil {
		canvas, _ := obsidiancanvas.Parse(canvasContent)
		nvs.PathMap[canvasPath] = &model.NoteView{
			Path:    canvasPath,
			Content: canvasContent,
			Canvas:  canvas,
		}
	}

	var sentMessages []string
	var lastMsgID int

	return &EnvMock{
		LatestNoteViewsFunc: func() *model.NoteViews { return nvs },
		LoggerFunc:          func() logger.Logger { return &logger.DummyLogger{} },
		BotIDFunc:           func() int64 { return 42 },
		GetTgUserCanvasStateFunc: func(_ context.Context, _ db.GetTgUserCanvasStateParams) (db.TgUserCanvasState, error) {
			return db.TgUserCanvasState{}, sql.ErrNoRows
		},
		UpsertTgUserCanvasStateFunc: func(_ context.Context, _ db.UpsertTgUserCanvasStateParams) error {
			return nil
		},
		DeleteTgUserCanvasStateFunc: func(_ context.Context, _ db.DeleteTgUserCanvasStateParams) error {
			return nil
		},
		UpsertTgUserCurrentHandlerFunc: func(_ context.Context, _ db.UpsertTgUserCurrentHandlerParams) error {
			return nil
		},
		SendMessageFunc: func(_ context.Context, _ int64, _, text, _ string) (int, error) {
			lastMsgID++
			sentMessages = append(sentMessages, text)
			return lastMsgID, nil
		},
		SendPhotoFunc: func(_ context.Context, _ int64, _, _, _, _ string) (int, error) {
			lastMsgID++
			return lastMsgID, nil
		},
		EditMessageTextFunc: func(_ context.Context, _ int64, _ int, _, _, _ string) error {
			return nil
		},
		EditMessageReplyMarkupFunc: func(_ context.Context, _ int64, _ int, _, _ string) error {
			return nil
		},
		DeleteMessageFunc: func(_ context.Context, _ int64, _ int, _ string) error {
			return nil
		},
		AnswerCallbackQueryFunc: func(_ context.Context, _, _ string, _ bool) error {
			return nil
		},
		RenderNoteHTMLFunc: func(nv *model.NoteView) (string, string) {
			return "rendered: " + nv.Path, ""
		},
	}
}

func TestResolve_StartCommand(t *testing.T) {
	canvasPath := "demo.canvas"
	env := newTestEnv(canvasPath, newTestCanvas())

	input := Input{
		Update: tgbotapi.Update{
			Message: &tgbotapi.Message{
				From: &tgbotapi.User{ID: 100},
				Chat: &tgbotapi.Chat{ID: 200},
				Text: "/start",
				Entities: []tgbotapi.MessageEntity{
					{Type: "bot_command", Offset: 0, Length: 6},
				},
			},
		},
		CanvasPath: canvasPath,
	}

	err := Resolve(context.Background(), env, input)
	require.NoError(t, err)

	// Should have sent a message (entry is "intro" due to single edge from START)
	require.NotEmpty(t, env.SendMessageFunc)
	calls := env.SendMessageCalls()
	require.GreaterOrEqual(t, len(calls), 1)
	require.Contains(t, calls[0].Text, "Welcome!")

	// Should have saved state
	stateCalls := env.UpsertTgUserCanvasStateCalls()
	require.Len(t, stateCalls, 1)
	require.Equal(t, "intro", stateCalls[0].P.CurrentNode)
}

func TestResolve_BrowseCommand(t *testing.T) {
	canvasPath := "test.canvas"
	env := newTestEnv(canvasPath, newTestCanvas())

	input := Input{
		Update: tgbotapi.Update{
			Message: &tgbotapi.Message{
				From: &tgbotapi.User{ID: 100},
				Chat: &tgbotapi.Chat{ID: 200},
				Text: "/browse",
				Entities: []tgbotapi.MessageEntity{
					{Type: "bot_command", Offset: 0, Length: 7},
				},
			},
		},
		CanvasPath: canvasPath,
	}

	err := Resolve(context.Background(), env, input)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(env.SendMessageCalls()), 1)
}

func TestResolve_CallbackOpen(t *testing.T) {
	canvasPath := "demo.canvas"
	env := newTestEnv(canvasPath, newTestCanvas())

	// Pre-set state: user is on "intro"
	env.GetTgUserCanvasStateFunc = func(_ context.Context, _ db.GetTgUserCanvasStateParams) (db.TgUserCanvasState, error) {
		return db.TgUserCanvasState{
			BotID:       42,
			UserID:      100,
			CanvasPath:  canvasPath,
			CurrentNode: "intro",
			Stack:       "[]",
			MessageID:   10,
		}, nil
	}

	input := Input{
		Update: tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID:   "cb1",
				From: &tgbotapi.User{ID: 100},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 200},
				},
				Data: "nav:open:page",
			},
		},
		CanvasPath: canvasPath,
	}

	err := Resolve(context.Background(), env, input)
	require.NoError(t, err)

	// Should have acknowledged callback
	require.Len(t, env.AnswerCallbackQueryCalls(), 1)

	// Should have saved state with "page" as current and "intro" in stack
	stateCalls := env.UpsertTgUserCanvasStateCalls()
	require.GreaterOrEqual(t, len(stateCalls), 1)
	last := stateCalls[len(stateCalls)-1]
	require.Equal(t, "page", last.P.CurrentNode)
	require.Contains(t, last.P.Stack, "intro")
}

func TestResolve_CallbackBack(t *testing.T) {
	canvasPath := "demo.canvas"
	env := newTestEnv(canvasPath, newTestCanvas())

	env.GetTgUserCanvasStateFunc = func(_ context.Context, _ db.GetTgUserCanvasStateParams) (db.TgUserCanvasState, error) {
		return db.TgUserCanvasState{
			BotID:       42,
			UserID:      100,
			CanvasPath:  canvasPath,
			CurrentNode: "page",
			Stack:       `["intro"]`,
			MessageID:   10,
		}, nil
	}

	input := Input{
		Update: tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID:   "cb2",
				From: &tgbotapi.User{ID: 100},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 200},
				},
				Data: "nav:back",
			},
		},
		CanvasPath: canvasPath,
	}

	err := Resolve(context.Background(), env, input)
	require.NoError(t, err)

	stateCalls := env.UpsertTgUserCanvasStateCalls()
	require.GreaterOrEqual(t, len(stateCalls), 1)
	last := stateCalls[len(stateCalls)-1]
	require.Equal(t, "intro", last.P.CurrentNode)
	require.Equal(t, "[]", last.P.Stack)
}

func TestResolve_CallbackExit(t *testing.T) {
	canvasPath := "demo.canvas"
	env := newTestEnv(canvasPath, newTestCanvas())

	env.GetTgUserCanvasStateFunc = func(_ context.Context, _ db.GetTgUserCanvasStateParams) (db.TgUserCanvasState, error) {
		return db.TgUserCanvasState{
			BotID:       42,
			UserID:      100,
			CanvasPath:  canvasPath,
			CurrentNode: "intro",
			Stack:       "[]",
			MessageID:   10,
		}, nil
	}

	input := Input{
		Update: tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID:   "cb3",
				From: &tgbotapi.User{ID: 100},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 200},
				},
				Data: "nav:exit",
			},
		},
		CanvasPath: canvasPath,
	}

	err := Resolve(context.Background(), env, input)
	require.NoError(t, err)

	// Should have deleted state
	require.Len(t, env.DeleteTgUserCanvasStateCalls(), 1)
	// Should have cleared handler
	require.Len(t, env.UpsertTgUserCurrentHandlerCalls(), 1)
	require.Equal(t, "", env.UpsertTgUserCurrentHandlerCalls()[0].P.Value)
	// Should have sent exit message
	sendCalls := env.SendMessageCalls()
	require.GreaterOrEqual(t, len(sendCalls), 1)
	require.Equal(t, "Exited browser.", sendCalls[len(sendCalls)-1].Text)
}

func TestResolve_NoCanvasFound(t *testing.T) {
	env := newTestEnv("missing.canvas", nil)

	input := Input{
		Update: tgbotapi.Update{
			Message: &tgbotapi.Message{
				From: &tgbotapi.User{ID: 100},
				Chat: &tgbotapi.Chat{ID: 200},
				Text: "/start",
				Entities: []tgbotapi.MessageEntity{
					{Type: "bot_command", Offset: 0, Length: 6},
				},
			},
		},
		CanvasPath: "missing.canvas",
	}

	err := Resolve(context.Background(), env, input)
	require.NoError(t, err)
	// No messages sent when canvas not found
	require.Empty(t, env.SendMessageCalls())
}

func TestResolve_NonCommandMessage(t *testing.T) {
	env := newTestEnv("demo.canvas", newTestCanvas())

	input := Input{
		Update: tgbotapi.Update{
			Message: &tgbotapi.Message{
				From: &tgbotapi.User{ID: 100},
				Chat: &tgbotapi.Chat{ID: 200},
				Text: "hello",
			},
		},
		CanvasPath: "demo.canvas",
	}

	err := Resolve(context.Background(), env, input)
	require.NoError(t, err)
	require.Empty(t, env.SendMessageCalls())
}
