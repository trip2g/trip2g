package createoidccredentials

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createoidccredentials_test . Env

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertOIDCCredentials(ctx context.Context, arg db.InsertOIDCCredentialsParams) (db.OidcCredential, error)
	DeactivateAllOIDCCredentials(ctx context.Context) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
	ValidateOIDCCredentials(ctx context.Context, issuer, clientID, clientSecret string) error
}

type Input = model.CreateOIDCCredentialsInput
type Payload = model.CreateOIDCCredentialsOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Name, ozzo.Required, ozzo.Length(1, 100)),
		ozzo.Field(&r.Issuer, ozzo.Required, is.URL),
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
	err = env.ValidateOIDCCredentials(ctx, input.Issuer, input.ClientID, input.ClientSecret)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Invalid credentials: %v", err)}, nil
	}

	encryptedSecret, err := env.EncryptData([]byte(input.ClientSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
	}

	err = env.DeactivateAllOIDCCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate existing credentials: %w", err)
	}

	scopes := "openid email profile"
	if input.Scopes != nil && *input.Scopes != "" {
		scopes = *input.Scopes
	}
	autoProvision := false
	if input.AutoProvision != nil {
		autoProvision = *input.AutoProvision
	}
	allowedEmailDomain := ""
	if input.AllowedEmailDomain != nil {
		allowedEmailDomain = *input.AllowedEmailDomain
	}
	requiredGroup := ""
	if input.RequiredGroup != nil {
		requiredGroup = *input.RequiredGroup
	}

	params := db.InsertOIDCCredentialsParams{
		Name:                  input.Name,
		Issuer:                input.Issuer,
		ClientID:              input.ClientID,
		ClientSecretEncrypted: encryptedSecret,
		Scopes:                scopes,
		AutoProvision:         autoProvision,
		AllowedEmailDomain:    allowedEmailDomain,
		RequiredGroup:         requiredGroup,
		Active:                true,
		CreatedBy:             int64(token.ID),
	}

	credentials, err := env.InsertOIDCCredentials(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "OIDC credentials with this client ID already exist"}, nil
		}
		return nil, fmt.Errorf("failed to insert oidc credentials: %w", err)
	}

	payload := model.CreateOIDCCredentialsPayload{
		Credentials: &credentials,
	}

	return &payload, nil
}
