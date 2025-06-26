package updatetgbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	GetTgBot(ctx context.Context, id int64) (db.TgBot, error)
	UpdateTgBot(ctx context.Context, arg db.UpdateTgBotParams) (db.TgBot, error)
}

func Resolve(ctx context.Context, env Env, input model.UpdateTgBotInput) (model.UpdateTgBotOrErrorPayload, error) {
	// Validate input
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.ID, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	// Check if bot exists
	_, err = env.GetTgBot(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Bot not found"}, nil
		}
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}

	// Update the bot
	bot, err := env.UpdateTgBot(ctx, db.UpdateTgBotParams{
		ID:          input.ID,
		Description: nullableString(input.Description),
		Enabled:     nullableBool(input.Enabled),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update telegram bot: %w", err)
	}

	return &model.UpdateTgBotPayload{
		TgBot: &bot,
	}, nil
}

func nullableString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullableBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}
