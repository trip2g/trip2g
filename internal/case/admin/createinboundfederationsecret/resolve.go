package createinboundfederationsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createinboundfederationsecret_test . Env

import (
	"context"
	"crypto/rand"
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

type Input = model.CreateInboundFederationSecretInput
type Payload = model.CreateInboundFederationSecretOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Kid, ozzo.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	secret, err := resolveSecret(input.SecretHex)
	if err != nil {
		return &model.ErrorPayload{Message: err.Error()}, nil //nolint:nilerr
	}

	encrypted, err := env.EncryptData(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	params := db.InsertFederationSecretParams{
		Kid:         input.Kid,
		SecretCrypt: encrypted,
		KbUrl:       nil,
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

	payload := model.CreateInboundFederationSecretPayload{
		ID:        row.ID,
		Kid:       row.Kid,
		SecretHex: hex.EncodeToString(secret),
	}

	return &payload, nil
}

// resolveSecret returns the 32-byte secret to store: caller-supplied hex if
// provided, otherwise crypto/rand. Hex must decode to exactly 32 bytes.
func resolveSecret(provided *string) ([]byte, error) {
	if provided == nil || *provided == "" {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, fmt.Errorf("failed to generate random secret: %w", err)
		}
		return secret, nil
	}

	decoded, err := hex.DecodeString(*provided)
	if err != nil {
		return nil, fmt.Errorf("secretHex is not valid hex: %w", err)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("secretHex must decode to exactly 32 bytes, got %d", len(decoded))
	}
	return decoded, nil
}
