package creategithuboauthcredentials

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg creategithuboauthcredentials_test . Env

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertGitHubOAuthCredentials(ctx context.Context, arg db.InsertGitHubOAuthCredentialsParams) (db.GithubOauthCredential, error)
	DeactivateAllGitHubOAuthCredentials(ctx context.Context) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
	ValidateGitHubOAuthCredentials(ctx context.Context, clientID, clientSecret string) error
}

type Input = model.CreateGitHubOAuthCredentialsInput
type Payload = model.CreateGitHubOAuthCredentialsOrErrorPayload

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
	err = env.ValidateGitHubOAuthCredentials(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Invalid credentials: %v", err)}, nil
	}

	encryptedSecret, err := env.EncryptData([]byte(input.ClientSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
	}

	err = env.DeactivateAllGitHubOAuthCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate existing credentials: %w", err)
	}

	params := db.InsertGitHubOAuthCredentialsParams{
		Name:                  input.Name,
		ClientID:              input.ClientID,
		ClientSecretEncrypted: encryptedSecret,
		Active:                true,
		CreatedBy:             int64(token.ID),
	}

	credentials, err := env.InsertGitHubOAuthCredentials(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "GitHub OAuth credentials with this client ID already exist"}, nil
		}
		return nil, fmt.Errorf("failed to insert github oauth credentials: %w", err)
	}

	payload := model.CreateGitHubOAuthCredentialsPayload{
		Credentials: &credentials,
	}

	return &payload, nil
}
