package createtgbot

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Env interface {
	InsertTgBot(ctx context.Context, arg db.InsertTgBotParams) (db.TgBot, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StartTgBot(ctx context.Context, id int64)
}

func Resolve(ctx context.Context, env Env, input model.CreateTgBotInput) (model.CreateTgBotOrErrorPayload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Validate input
	err = ozzo.ValidateStruct(&input,
		ozzo.Field(&input.Token, ozzo.Required),
		ozzo.Field(&input.Description, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	// Validate token format (basic check for bot token)
	if len(input.Token) < 40 || !containsColon(input.Token) {
		return &model.ErrorPayload{Message: "Invalid bot token format"}, nil
	}

	// Validate token with real Telegram API and get bot info
	botAPI, err := tgbotapi.NewBotAPI(input.Token)
	if err != nil {
		return &model.ErrorPayload{Message: "Invalid bot token or API error"}, nil //nolint:nilerr // error is handled by returning ErrorPayload
	}

	// Get bot name from API
	botName := botAPI.Self.UserName
	if botName == "" {
		return &model.ErrorPayload{Message: "Could not retrieve bot username from Telegram API"}, nil
	}

	// Create the bot
	bot, err := env.InsertTgBot(ctx, db.InsertTgBotParams{
		Token:       input.Token,
		Name:        botName,
		Description: input.Description,
		CreatedBy:   int64(token.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	env.StartTgBot(ctx, bot.ID)

	return &model.CreateTgBotPayload{
		TgBot: &bot,
	}, nil
}

func containsColon(s string) bool {
	for _, r := range s {
		if r == ':' {
			return true
		}
	}
	return false
}
