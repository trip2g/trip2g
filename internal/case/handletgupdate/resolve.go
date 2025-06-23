package handletgupdate

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Env interface {
	InsertTgUserProfile(ctx context.Context, arg db.InsertTgUserProfileParams) error
	SendMessage(userID int64, text string) (tgbotapi.Message, error)
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func Resolve(ctx context.Context, env Env, update tgbotapi.Update) error {
	// Update user profile if we have a message with user info
	if update.Message != nil && update.Message.From != nil {
		profileParams := db.InsertTgUserProfileParams{
			ChatID:    update.Message.Chat.ID,
			FirstName: toNullString(update.Message.From.FirstName),
			LastName:  toNullString(update.Message.From.LastName),
			Username:  toNullString(update.Message.From.UserName),
		}

		err := env.InsertTgUserProfile(ctx, profileParams)
		if err != nil {
			return fmt.Errorf("failed to insert user profile: %w", err)
		}
	}

	_, err := env.SendMessage(update.Message.Chat.ID, "Hello, World!")
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
