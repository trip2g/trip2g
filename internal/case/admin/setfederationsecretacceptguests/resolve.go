package setfederationsecretacceptguests

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg setfederationsecretacceptguests_test . Env

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	FederationSecretByID(ctx context.Context, id int64) (db.FederationSecret, error)
	SetFederationSecretAcceptGuests(ctx context.Context, arg db.SetFederationSecretAcceptGuestsParams) error
}

type Input = model.SetFederationSecretAcceptGuestsInput
type Payload = model.SetFederationSecretAcceptGuestsOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	row, err := env.FederationSecretByID(ctx, input.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return &model.ErrorPayload{Message: "Federation secret not found"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get federation secret: %w", err)
	}
	if row.KbUrl != nil {
		return &model.ErrorPayload{Message: "Only an inbound key can accept people"}, nil
	}
	if row.RevokedAt != nil {
		return &model.ErrorPayload{Message: "The key is revoked"}, nil
	}

	params := db.SetFederationSecretAcceptGuestsParams{ID: row.ID, AcceptGuests: input.AcceptGuests}
	err = env.SetFederationSecretAcceptGuests(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to set accept guests: %w", err)
	}

	return &model.SetFederationSecretAcceptGuestsPayload{ID: row.ID, AcceptGuests: input.AcceptGuests}, nil
}
