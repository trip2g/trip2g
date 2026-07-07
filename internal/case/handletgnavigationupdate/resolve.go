package handletgnavigationupdate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

const handlerNavigation = "navigation"

type Env interface {
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	BotID() int64
	BotLink() string
	Logger() logger.Logger
	LatestNoteViews() *model.NoteViews
	TgUserStateByBotIDAndChatID(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error)
	UpsertTgUserState(ctx context.Context, arg db.UpsertTgUserStateParams) error
}

type navState struct {
	Current   int64   `json:"current,omitempty"`
	Stack     []int64 `json:"stack"`
	LastMsgID int64   `json:"last_msg_id,omitempty"`
}

// stateFragment is what we read/write in tg_user_states.data.
// We merge only our keys to preserve unrelated state owned by other handlers.
type stateFragment struct {
	Handler string    `json:"handler,omitempty"`
	Nav     *navState `json:"nav,omitempty"`
}

// ResolveOpen opens a specific note by pathID, saving nav state and sending the note.
// Used for deep links: t.me/bot?start=browse_<pathID>.
func ResolveOpen(ctx context.Context, env Env, update tgbotapi.Update, pathID int64) error {
	chatID := extractChatID(update)
	if chatID == 0 {
		return nil
	}
	rawState, frag, err := loadState(ctx, env, chatID)
	if err != nil {
		rawState = "{}"
		frag = &stateFragment{}
	}
	if frag.Nav == nil {
		frag.Nav = &navState{}
	}
	if frag.Nav.Current != 0 {
		frag.Nav.Stack = append(frag.Nav.Stack, frag.Nav.Current)
	}
	frag.Nav.Current = pathID
	frag.Nav.LastMsgID = 0 // deep link always opens a new message
	frag.Handler = handlerNavigation
	if saveErr := saveState(ctx, env, chatID, rawState, frag); saveErr != nil {
		env.Logger().Error("nav: save state on deep link", "error", saveErr)
	}
	return sendNote(ctx, env, chatID, pathID, frag, rawState)
}

func Resolve(ctx context.Context, env Env, update tgbotapi.Update) error {
	chatID := extractChatID(update)
	if chatID == 0 {
		return nil
	}

	rawState, frag, err := loadState(ctx, env, chatID)
	if err != nil {
		env.Logger().Error("nav: failed to load state", "error", err)
		rawState = "{}"
		frag = &stateFragment{}
	}
	if frag.Nav == nil {
		frag.Nav = &navState{}
	}

	if update.CallbackQuery != nil {
		return handleCallback(ctx, env, update.CallbackQuery, chatID, rawState, frag)
	}

	if update.Message != nil {
		return handleMessage(ctx, env, update.Message, chatID, rawState, frag)
	}

	return nil
}

func handleCallback(ctx context.Context, env Env, cb *tgbotapi.CallbackQuery, chatID int64, rawState string, frag *stateFragment) error {
	// Acknowledge immediately
	ack := tgbotapi.NewCallback(cb.ID, "")
	_, _ = env.Request(ack)

	parts := strings.SplitN(cb.Data, ":", 3)
	if len(parts) < 2 || parts[0] != "nav" {
		return nil
	}

	switch parts[1] {
	case "open":
		if len(parts) < 3 {
			return nil
		}
		pathID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return nil //nolint:nilerr // malformed callback data: ignore stale/invalid button
		}
		// Push current note onto stack before navigating
		if frag.Nav.Current != 0 {
			frag.Nav.Stack = append(frag.Nav.Stack, frag.Nav.Current)
		}
		frag.Nav.Current = pathID
		if saveErr := saveState(ctx, env, chatID, rawState, frag); saveErr != nil {
			env.Logger().Error("nav: save state", "error", saveErr)
		}
		return sendNote(ctx, env, chatID, pathID, frag, rawState)

	case "back":
		if len(frag.Nav.Stack) == 0 {
			msg := tgbotapi.NewMessage(chatID, "Already at the top.")
			_, _ = env.Send(msg)
			return nil
		}
		prevID := frag.Nav.Stack[len(frag.Nav.Stack)-1]
		frag.Nav.Stack = frag.Nav.Stack[:len(frag.Nav.Stack)-1]
		frag.Nav.Current = prevID
		if err := saveState(ctx, env, chatID, rawState, frag); err != nil {
			env.Logger().Error("nav: save state", "error", err)
		}
		return sendNote(ctx, env, chatID, prevID, frag, rawState)

	case "exit":
		frag.Handler = ""
		frag.Nav = nil
		if err := saveState(ctx, env, chatID, rawState, frag); err != nil {
			env.Logger().Error("nav: save state on exit", "error", err)
		}
		msg := tgbotapi.NewMessage(chatID, "Exited browser.")
		_, _ = env.Send(msg)
		return nil
	}

	return nil
}

func handleMessage(ctx context.Context, env Env, msg *tgbotapi.Message, chatID int64, rawState string, frag *stateFragment) error {
	if msg.IsCommand() {
		switch msg.Command() {
		case "browse", "start":
			return handleBrowseCommand(ctx, env, msg, chatID, rawState, frag)
		}
	}

	reply := tgbotapi.NewMessage(chatID, "Use the buttons to navigate. /browse to restart.")
	_, _ = env.Send(reply)
	return nil
}

// handleBrowseCommand handles the /browse and /start commands.
func handleBrowseCommand(ctx context.Context, env Env, msg *tgbotapi.Message, chatID int64, rawState string, frag *stateFragment) error {
	if pathID, ok := parseBrowseDeepLink(msg.CommandArguments()); ok {
		if frag.Nav == nil {
			frag.Nav = &navState{}
		}
		if frag.Nav.Current != 0 {
			frag.Nav.Stack = append(frag.Nav.Stack, frag.Nav.Current)
		}
		frag.Nav.Current = pathID
		frag.Nav.LastMsgID = 0 // deep link always opens a new message
		frag.Handler = handlerNavigation
		if saveErr := saveState(ctx, env, chatID, rawState, frag); saveErr != nil {
			env.Logger().Error("nav: save state on deep link", "error", saveErr)
		}
		return sendNote(ctx, env, chatID, pathID, frag, rawState)
	}

	note := FindStartNote(env.LatestNoteViews())
	if note == nil {
		reply := tgbotapi.NewMessage(chatID, "No _bot_start.md found in vault.")
		_, _ = env.Send(reply)
		return nil
	}
	frag.Nav = &navState{Current: note.PathID}
	frag.Handler = handlerNavigation
	if saveErr := saveState(ctx, env, chatID, rawState, frag); saveErr != nil {
		env.Logger().Error("nav: save state on browse", "error", saveErr)
	}
	return sendNote(ctx, env, chatID, note.PathID, frag, rawState)
}

// parseBrowseDeepLink extracts the pathID from a "browse_<pathID>" command argument.
func parseBrowseDeepLink(args string) (int64, bool) {
	if !strings.HasPrefix(args, "browse_") {
		return 0, false
	}
	pathID, err := strconv.ParseInt(args[7:], 10, 64)
	if err != nil {
		return 0, false
	}
	return pathID, true
}

func sendNote(ctx context.Context, env Env, chatID, pathID int64, frag *stateFragment, rawState string) error {
	text, keyboard, err := RenderNote(env.LatestNoteViews(), pathID, frag.Nav.Stack, env.BotLink())
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Note not found (%d).", pathID))
		_, _ = env.Send(msg)
		return nil //nolint:nilerr // render failure is reported to the user; don't fail the update
	}

	if frag.Nav.LastMsgID != 0 {
		edit := tgbotapi.NewEditMessageText(chatID, int(frag.Nav.LastMsgID), text)
		edit.ParseMode = tgbotapi.ModeHTML
		edit.ReplyMarkup = keyboard
		_, _ = env.Send(edit)
		return nil
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard
	sent, sendErr := env.Send(msg)
	if sendErr == nil {
		frag.Nav.LastMsgID = int64(sent.MessageID)
		_ = saveState(ctx, env, chatID, rawState, frag)
	}
	return sendErr
}

func loadState(ctx context.Context, env Env, chatID int64) (string, *stateFragment, error) {
	state, dbErr := env.TgUserStateByBotIDAndChatID(ctx, db.TgUserStateByBotIDAndChatIDParams{
		BotID:  env.BotID(),
		ChatID: chatID,
	})
	if dbErr != nil {
		if errors.Is(dbErr, sql.ErrNoRows) {
			return "{}", &stateFragment{}, nil
		}
		return "{}", &stateFragment{}, dbErr
	}
	var f stateFragment
	if jsonErr := json.Unmarshal([]byte(state.Data), &f); jsonErr != nil {
		// Corrupt state: keep raw data but reset our fragment (graceful degradation).
		return state.Data, &stateFragment{}, nil //nolint:nilerr // intentionally reset on unparsable state
	}
	return state.Data, &f, nil
}

func saveState(ctx context.Context, env Env, chatID int64, rawJSON string, frag *stateFragment) error {
	// Merge our keys into the existing JSON to preserve other state owned by other handlers.
	var full map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &full); err != nil {
		full = make(map[string]json.RawMessage)
	}

	if frag.Handler == "" {
		delete(full, "handler")
	} else {
		b, _ := json.Marshal(frag.Handler)
		full["handler"] = b
	}

	if frag.Nav == nil {
		delete(full, "nav")
	} else {
		b, _ := json.Marshal(frag.Nav)
		full["nav"] = b
	}

	merged, err := json.Marshal(full)
	if err != nil {
		return fmt.Errorf("marshal nav state: %w", err)
	}

	// Read current state for UpdateCount and Value
	cur, dbErr := env.TgUserStateByBotIDAndChatID(ctx, db.TgUserStateByBotIDAndChatIDParams{
		BotID:  env.BotID(),
		ChatID: chatID,
	})
	value := "pending"
	var updateCount int64
	if dbErr == nil {
		value = cur.Value
		updateCount = cur.UpdateCount
	}

	return env.UpsertTgUserState(ctx, db.UpsertTgUserStateParams{
		BotID:       env.BotID(),
		ChatID:      chatID,
		Value:       value,
		Data:        string(merged),
		UpdateCount: updateCount + 1,
	})
}

func extractChatID(update tgbotapi.Update) int64 {
	switch {
	case update.Message != nil:
		return update.Message.Chat.ID
	case update.CallbackQuery != nil:
		return update.CallbackQuery.Message.Chat.ID
	default:
		return 0
	}
}
