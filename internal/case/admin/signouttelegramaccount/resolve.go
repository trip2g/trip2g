package signouttelegramaccount

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
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
		Enabled:     ptr.To(int64(0)),
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
