package createoutboundfederationsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createoutboundfederationsecret_test . Env

import (
	"context"
	"encoding/hex"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertFederationSecret(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
}

type Input = model.CreateOutboundFederationSecretInput
type Payload = model.CreateOutboundFederationSecretOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Kid, ozzo.Required),
		ozzo.Field(&r.SecretHex, ozzo.Required),
		ozzo.Field(&r.KbURL, ozzo.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	secret, err := hex.DecodeString(input.SecretHex)
	if err != nil || len(secret) != 32 { //nolint:nilerr // intentional: convert decode error to user-facing payload
		return &model.ErrorPayload{Message: "secretHex must be exactly 32 bytes (64 hex characters)"}, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	encrypted, err := env.EncryptData(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	kbURL := input.KbURL
	params := db.InsertFederationSecretParams{
		Kid:         input.Kid,
		SecretCrypt: encrypted,
		KbUrl:       &kbURL,
		Description: input.Description,
		CreatedBy:   int64(token.ID),
	}

	row, err := env.InsertFederationSecret(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "Federation secret with this kid already exists"}, nil
		}
		return nil, fmt.Errorf("failed to insert federation secret: %w", err)
	}

	payload := model.CreateOutboundFederationSecretPayload{
		ID:  row.ID,
		Kid: row.Kid,
	}

	return &payload, nil
}
