package deletetelegramaccount

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
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
	DeleteTelegramAccount(ctx context.Context, id int64) error
}

type Input = model.AdminDeleteTelegramAccountInput
type Payload = model.AdminDeleteTelegramAccountOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.ID, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	// Check if account exists
	_, err = env.GetTelegramAccountByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Account not found"}, nil
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Delete the account
	err = env.DeleteTelegramAccount(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete telegram account: %w", err)
	}

	payload := model.AdminDeleteTelegramAccountPayload{
		Success: true,
	}

	return &payload, nil
}
