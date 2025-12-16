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

// Note: sql import is still used for sql.ErrNoRows

type Env interface {
	TgBot(ctx context.Context, id int64) (db.TgBot, error)
	UpdateTgBot(ctx context.Context, arg db.UpdateTgBotParams) (db.TgBot, error)
	StartTgBot(ctx context.Context, id int64)
	StopTgBot(ctx context.Context, id int64)
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
	_, err = env.TgBot(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Bot not found"}, nil
		}
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}

	// Update the bot
	bot, err := env.UpdateTgBot(ctx, db.UpdateTgBotParams{
		ID:          input.ID,
		Description: input.Description,
		Enabled:     input.Enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update telegram bot: %w", err)
	}

	if input.Enabled != nil {
		if *input.Enabled {
			env.StartTgBot(ctx, input.ID)
		} else {
			env.StopTgBot(ctx, input.ID)
		}
	}

	return &model.UpdateTgBotPayload{
		TgBot: &bot,
	}, nil
}
