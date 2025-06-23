package handletgupdate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Env interface {
	TgUserStateByBotIDAndChatID(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error)
	InsertTgUserProfile(ctx context.Context, arg db.InsertTgUserProfileParams) error
	UpsertTgUserState(ctx context.Context, arg db.UpsertTgUserStateParams) error
	SendMessage(userID int64, text string) (tgbotapi.Message, error)
	CalculateSha256(s string) string
	LatestNoteViews() *model.NoteViews // TODO: read LiveNoteViews for production users
	BotID() int64
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

	_, err := env.SendMessage(update.Message.Chat.ID, "Hello, World!")
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	userState, err := getUserState(ctx, env, update.Message.Chat.ID)
	if err != nil {
		return fmt.Errorf("failed to get user state: %w", err)
	}

	err = updateUserState(ctx, env, *userState)
	if err != nil {
		return fmt.Errorf("failed to update user state: %w", err)
	}

	return nil
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
