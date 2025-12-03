package updatetelegramaccount

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
	UpdateTelegramAccount(ctx context.Context, arg db.UpdateTelegramAccountParams) error
}

type Input = model.AdminUpdateTelegramAccountInput
type Payload = model.AdminUpdateTelegramAccountOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.ID, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	// Check if account exists
	account, err := env.GetTelegramAccountByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Account not found"}, nil
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Update the account
	params := db.UpdateTelegramAccountParams{
		ID:          input.ID,
		DisplayName: nullableString(input.DisplayName),
		Enabled:     nullableBoolToInt64(input.Enabled),
	}

	err = env.UpdateTelegramAccount(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update telegram account: %w", err)
	}

	// Refetch the account to return updated data
	account, err = env.GetTelegramAccountByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated account: %w", err)
	}

	payload := model.AdminUpdateTelegramAccountPayload{
		Account: &account,
	}

	return &payload, nil
}

func nullableString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullableBoolToInt64(b *bool) sql.NullInt64 {
	if b == nil {
		return sql.NullInt64{}
	}
	v := int64(0)
	if *b {
		v = 1
	}
	return sql.NullInt64{Int64: v, Valid: true}
}
