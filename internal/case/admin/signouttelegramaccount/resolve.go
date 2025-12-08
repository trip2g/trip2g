package signouttelegramaccount

import (
	"context"
	"database/sql"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	UpdateTelegramAccount(ctx context.Context, arg db.UpdateTelegramAccountParams) error
}

type Input = model.AdminSignOutTelegramAccountInput
type Payload = model.AdminSignOutTelegramAccountOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Set enabled = false and clear session_data
	err := env.UpdateTelegramAccount(ctx, db.UpdateTelegramAccountParams{
		ID:          input.ID,
		Enabled:     sql.NullInt64{Int64: 0, Valid: true},
		SessionData: []byte{}, // Clear encrypted session data
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign out telegram account: %w", err)
	}

	payload := model.AdminSignOutTelegramAccountPayload{
		Success: true,
	}

	return &payload, nil
}
