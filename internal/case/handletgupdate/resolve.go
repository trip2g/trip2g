package handletgupdate

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Env interface {
	SendMessage(userID int64, text string) (tgbotapi.Message, error)
}

func Resolve(ctx context.Context, env Env, update tgbotapi.Update) error {
	_, err := env.SendMessage(update.Message.Chat.ID, "Hello, World!")
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
