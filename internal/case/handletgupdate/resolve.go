package handletgupdate

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Env interface {
	TgUserStateByBotIDAndChatID(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error)
	InsertTgUserProfile(ctx context.Context, arg db.InsertTgUserProfileParams) error
	UpsertTgUserState(ctx context.Context, arg db.UpsertTgUserStateParams) error
	CalculateSha256(s string) string
	LatestNoteViews() *model.NoteViews // TODO: read LiveNoteViews for production users
	Logger() logger.Logger
	BotID() int64
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

type Question struct {
	ID       int
	Text     string
	Category string
}

type UserStateData struct {
}

type UserState struct {
	UserStateData
	ChatID int64
	Value  string

	UpdateCount int64
}

func Resolve(ctx context.Context, env Env, update tgbotapi.Update) error {
	// Update user profile if we have a message with user info
	if update.Message != nil && update.Message.From != nil {
		profileParams := db.InsertTgUserProfileParams{
			ChatID:    update.Message.Chat.ID,
			BotID:     env.BotID(),
			FirstName: toNullString(update.Message.From.FirstName),
			LastName:  toNullString(update.Message.From.LastName),
			Username:  toNullString(update.Message.From.UserName),
		}

		hashValue, err := json.Marshal(profileParams)
		if err != nil {
			return fmt.Errorf("failed to marshal user profile params: %w", err)
		}

		profileParams.Sha256Hash = env.CalculateSha256(string(hashValue))

		err = env.InsertTgUserProfile(ctx, profileParams)
		if err != nil {
			return fmt.Errorf("failed to insert user profile: %w", err)
		}
	}

	var chatID int64

	if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else if update.Message != nil {
		chatID = update.Message.Chat.ID
	}

	userState, err := getUserState(ctx, env, chatID)
	if err != nil {
		return fmt.Errorf("failed to get user state: %w", err)
	}

	if update.CallbackQuery != nil {
		return handleCallbackQuery(ctx, env, update, userState)
	}

	if update.Message != nil && update.Message.IsCommand() {
		return handleCommands(ctx, env, update, userState)
	}

	getQuestiens(ctx, env)

	err = updateUserState(ctx, env, *userState)
	if err != nil {
		return fmt.Errorf("failed to update user state: %w", err)
	}

	return nil
}

func handleCallbackQuery(ctx context.Context, env Env, update tgbotapi.Update, userState *UserState) error {
	switch update.CallbackQuery.Data {
	case "start_mbti":
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)

		_, err := env.Request(callback)
		if err != nil {
			return fmt.Errorf("failed to send callback: %w", err)
		}

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Starting MBTI test...")

		_, err = env.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

	default:
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Unknown action")

		_, err := env.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func handleCommands(ctx context.Context, env Env, update tgbotapi.Update, userState *UserState) error {
	switch update.Message.Command() {
	case "start":
		return sendStartMenu(ctx, env, update)

	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")

		_, err := env.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func sendStartMenu(ctx context.Context, env Env, update tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Welcome to Trip2G!")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Начать тест", "start_mbti"),
			tgbotapi.NewInlineKeyboardButtonData("Подробнее", "more_details"),
		),
	)

	_, err := env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send start menu: %w", err)
	}

	return nil
}

func getQuestiens(ctx context.Context, env Env) ([]Question, error) {
	notes := env.LatestNoteViews()

	re := regexp.MustCompile(`_mbti/(\d+)\.md$`)

	var res []Question

	for _, note := range notes.List {
		m := re.FindStringSubmatch(note.Path)
		if len(m) < 2 {
			continue
		}

		id, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse question ID from path %s: %w", note.Path, err)
		}

		categoryI, ok := note.RawMeta["category"]
		if !ok {
			return nil, fmt.Errorf("category not found in note %s", note.Path)
		}

		category, ok := categoryI.(string)
		if !ok {
			return nil, fmt.Errorf("category is not a string in note %s. %T", note.Path, categoryI)
		}

		question := Question{
			ID:       id,
			Text:     trimYAMLFrontMatter(string(note.Content)),
			Category: category,
		}

		res = append(res, question)
	}

	slices.SortFunc(res, func(a, b Question) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return res, nil
}

func trimYAMLFrontMatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return content
	}

	return strings.TrimSpace(parts[2])
}

func getUserState(ctx context.Context, env Env, chatID int64) (*UserState, error) {
	params := db.TgUserStateByBotIDAndChatIDParams{
		BotID:  env.BotID(),
		ChatID: chatID,
	}

	userState := UserState{
		ChatID: chatID,
		Value:  "pending", // Default value if no state found
	}

	row, err := env.TgUserStateByBotIDAndChatID(ctx, params)
	if err != nil {
		if db.IsNoFound(err) {
			return &userState, nil
		}

		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	userState.Value = row.Value

	err = json.Unmarshal([]byte(row.Data), &userState.UserStateData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state data: %w", err)
	}

	return &userState, nil
}

func updateUserState(ctx context.Context, env Env, state UserState) error {
	data, err := json.Marshal(state.UserStateData)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	upsertParams := db.UpsertTgUserStateParams{
		ChatID: state.ChatID,
		BotID:  env.BotID(),
		Value:  state.Value,
		Data:   string(data),

		UpdateCount: state.UpdateCount + 1,
	}

	err = env.UpsertTgUserState(ctx, upsertParams)
	if err != nil {
		return fmt.Errorf("failed to upsert user state: %w", err)
	}

	return nil
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
