package handletgcanvasupdate

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/obsidiancanvas"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

// Env declares the dependencies for canvas navigation handling.
type Env interface {
	LatestNoteViews() *model.NoteViews
	Logger() logger.Logger
	BotID() int64

	GetTgUserCanvasState(ctx context.Context, p db.GetTgUserCanvasStateParams) (db.TgUserCanvasState, error)
	UpsertTgUserCanvasState(ctx context.Context, p db.UpsertTgUserCanvasStateParams) error
	DeleteTgUserCanvasState(ctx context.Context, p db.DeleteTgUserCanvasStateParams) error
	UpsertTgUserCurrentHandler(ctx context.Context, p db.UpsertTgUserCurrentHandlerParams) error

	SendMessage(ctx context.Context, chatID int64, bcID, text, markup string) (messageID int, err error)
	SendPhoto(ctx context.Context, chatID int64, bcID, mediaPath, caption, markup string) (messageID int, err error)
	EditMessageText(ctx context.Context, chatID int64, messageID int, bcID, text, markup string) error
	EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int, bcID, markup string) error
	DeleteMessage(ctx context.Context, chatID int64, messageID int, bcID string) error
	AnswerCallbackQuery(ctx context.Context, callbackID, text string, alert bool) error

	RenderNoteHTML(nv *model.NoteView) (text, firstMedia string)
}

// Input is passed by the dispatcher to Resolve.
type Input struct {
	Update     tgbotapi.Update
	CanvasPath string // vault-relative path of the .canvas file
}

// Resolve is the main entry point for canvas navigation updates.
func Resolve(ctx context.Context, env Env, input Input) error {
	log := logger.WithPrefix(env.Logger(), "canvas")

	if input.Update.CallbackQuery != nil {
		return handleCallback(ctx, env, log, input)
	}
	if input.Update.Message != nil {
		return handleMessage(ctx, env, log, input)
	}
	return nil
}

func handleMessage(ctx context.Context, env Env, log logger.Logger, input Input) error {
	msg := input.Update.Message
	if !msg.IsCommand() {
		return nil
	}

	switch msg.Command() {
	case "start", "browse":
		return handleStart(ctx, env, log, input)
	}
	return nil
}

func handleStart(ctx context.Context, env Env, log logger.Logger, input Input) error {
	canvas := loadCanvas(env, input.CanvasPath)
	if canvas == nil {
		log.Error("canvas not available", "path", input.CanvasPath)
		return nil
	}

	entryID := canvas.Entry()
	if entryID == "" {
		log.Error("canvas has no START node", "path", input.CanvasPath)
		return nil
	}

	chatID := extractChatID(input.Update)
	bcID := extractBusinessConnectionID(input.Update)
	userID := extractUserID(input.Update)

	// Render entry node
	text, media, markup := renderNode(env, canvas, entryID)

	// Send message
	var messageID int
	if media != "" {
		messageID, _ = env.SendPhoto(ctx, chatID, bcID, media, text, markup)
	} else {
		messageID, _ = env.SendMessage(ctx, chatID, bcID, text, markup)
	}

	// Save initial state
	saveState(ctx, env, log, canvasState{
		BotID:      env.BotID(),
		BCID:       bcID,
		UserID:     userID,
		CanvasPath: input.CanvasPath,
		Current:    entryID,
		Stack:      nil,
		LastMedia:  media,
		MessageID:  int64(messageID),
	})

	return nil
}

func handleCallback(ctx context.Context, env Env, log logger.Logger, input Input) error {
	cb := input.Update.CallbackQuery
	data := cb.Data

	// Acknowledge immediately
	_ = env.AnswerCallbackQuery(ctx, cb.ID, "", false)

	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 || parts[0] != "nav" {
		return nil
	}

	chatID := extractChatID(input.Update)
	bcID := extractBusinessConnectionID(input.Update)
	userID := extractUserID(input.Update)

	state, err := loadState(ctx, env, userID, bcID)
	if err != nil {
		log.Error("canvas: load state failed", "error", err)
		return nil
	}

	canvas := loadCanvas(env, input.CanvasPath)
	if canvas == nil {
		log.Error("canvas not available", "path", input.CanvasPath)
		return nil
	}

	switch parts[1] {
	case "open":
		if len(parts) < 3 {
			return nil
		}
		targetID := parts[2]
		if _, ok := canvas.Node(targetID); !ok {
			return nil
		}

		// Push current onto stack
		if state.Current != "" {
			state.Stack = append(state.Stack, state.Current)
		}
		state.Current = targetID

		text, media, markup := renderNode(env, canvas, targetID)
		newMsgID := renderTransition(ctx, env, chatID, bcID, int(state.MessageID), state.LastMedia, media, text, markup)
		state.MessageID = int64(newMsgID)
		state.LastMedia = media

		saveState(ctx, env, log, canvasState{
			BotID:      env.BotID(),
			BCID:       bcID,
			UserID:     userID,
			CanvasPath: input.CanvasPath,
			Current:    state.Current,
			Stack:      state.Stack,
			LastMedia:  state.LastMedia,
			MessageID:  state.MessageID,
		})

	case "back":
		if len(state.Stack) == 0 {
			return nil
		}
		prevID := state.Stack[len(state.Stack)-1]
		state.Stack = state.Stack[:len(state.Stack)-1]
		state.Current = prevID

		text, media, markup := renderNode(env, canvas, prevID)
		newMsgID := renderTransition(ctx, env, chatID, bcID, int(state.MessageID), state.LastMedia, media, text, markup)
		state.MessageID = int64(newMsgID)
		state.LastMedia = media

		saveState(ctx, env, log, canvasState{
			BotID:      env.BotID(),
			BCID:       bcID,
			UserID:     userID,
			CanvasPath: input.CanvasPath,
			Current:    state.Current,
			Stack:      state.Stack,
			LastMedia:  state.LastMedia,
			MessageID:  state.MessageID,
		})

	case "exit":
		// Delete state and clear handler
		_ = env.DeleteTgUserCanvasState(ctx, db.DeleteTgUserCanvasStateParams{
			BotID:                env.BotID(),
			BusinessConnectionID: bcID,
			UserID:               userID,
		})
		_ = env.UpsertTgUserCurrentHandler(ctx, db.UpsertTgUserCurrentHandlerParams{
			BotID:                env.BotID(),
			BusinessConnectionID: bcID,
			UserID:               userID,
			Value:                "",
		})
		_, _ = env.SendMessage(ctx, chatID, bcID, "Exited browser.", "")
	}

	return nil
}

func loadCanvas(env Env, canvasPath string) *obsidiancanvas.Canvas {
	nvs := env.LatestNoteViews()
	if nvs == nil {
		return nil
	}
	nv, ok := nvs.PathMap[canvasPath]
	if !ok || nv == nil {
		return nil
	}
	return nv.Canvas
}

func extractChatID(update tgbotapi.Update) int64 {
	switch {
	case update.CallbackQuery != nil && update.CallbackQuery.Message != nil:
		return update.CallbackQuery.Message.Chat.ID
	case update.Message != nil:
		return update.Message.Chat.ID
	default:
		return 0
	}
}

func extractUserID(update tgbotapi.Update) int64 {
	switch {
	case update.CallbackQuery != nil:
		return update.CallbackQuery.From.ID
	case update.Message != nil:
		return update.Message.From.ID
	default:
		return 0
	}
}

func extractBusinessConnectionID(_ tgbotapi.Update) string {
	// The standard tgbotapi v5 library doesn't expose BusinessConnectionID.
	// In production the dispatcher passes it via Input or loads from DB state.
	// For now this returns "" (direct bot chats). Business connection support
	// will be wired in the dispatcher follow-up step.
	return ""
}

func loadState(ctx context.Context, env Env, userID int64, bcID string) (*canvasState, error) {
	dbState, err := env.GetTgUserCanvasState(ctx, db.GetTgUserCanvasStateParams{
		BotID:                env.BotID(),
		BusinessConnectionID: bcID,
		UserID:               userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &canvasState{}, nil
		}
		return nil, err
	}
	stack := deserializeStack(dbState.Stack)
	return &canvasState{
		BotID:      dbState.BotID,
		BCID:       dbState.BusinessConnectionID,
		UserID:     dbState.UserID,
		CanvasPath: dbState.CanvasPath,
		Current:    dbState.CurrentNode,
		Stack:      stack,
		LastMedia:  dbState.LastMedia,
		MessageID:  dbState.MessageID,
	}, nil
}
