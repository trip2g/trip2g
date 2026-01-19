package creategoogleoauthcredentials

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg creategoogleoauthcredentials_test . Env

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertGoogleOAuthCredentials(ctx context.Context, arg db.InsertGoogleOAuthCredentialsParams) (db.GoogleOauthCredential, error)
	DeactivateAllGoogleOAuthCredentials(ctx context.Context) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
	ValidateGoogleOAuthCredentials(ctx context.Context, clientID, clientSecret string) error
}

type Input = model.CreateGoogleOAuthCredentialsInput
type Payload = model.CreateGoogleOAuthCredentialsOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Name, ozzo.Required, ozzo.Length(1, 100)),
		ozzo.Field(&r.ClientID, ozzo.Required, ozzo.Length(10, 200)),
		ozzo.Field(&r.ClientSecret, ozzo.Required, ozzo.Length(10, 200)),
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

	// Validate credentials before saving
	err = env.ValidateGoogleOAuthCredentials(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Invalid credentials: %v", err)}, nil
	}

	encryptedSecret, err := env.EncryptData([]byte(input.ClientSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
	}

	err = env.DeactivateAllGoogleOAuthCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate existing credentials: %w", err)
	}

	params := db.InsertGoogleOAuthCredentialsParams{
		Name:                  input.Name,
		ClientID:              input.ClientID,
		ClientSecretEncrypted: encryptedSecret,
		Active:                true,
		CreatedBy:             int64(token.ID),
	}

	credentials, err := env.InsertGoogleOAuthCredentials(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "Google OAuth credentials with this client ID already exist"}, nil
		}
		return nil, fmt.Errorf("failed to insert google oauth credentials: %w", err)
	}

	payload := model.CreateGoogleOAuthCredentialsPayload{
		Credentials: &credentials,
	}

	return &payload, nil
}
