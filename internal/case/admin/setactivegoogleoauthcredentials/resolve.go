package setactivegoogleoauthcredentials

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	SetActiveGoogleOAuthCredentials(ctx context.Context, id int64) error
	GetGoogleOAuthCredentials(ctx context.Context, id int64) (db.GoogleOauthCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.SetActiveGoogleOAuthCredentialsInput
type Payload = model.SetActiveGoogleOAuthCredentialsOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.ID, ozzo.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	err = env.SetActiveGoogleOAuthCredentials(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to set active google oauth credentials: %w", err)
	}

	credentials, err := env.GetGoogleOAuthCredentials(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get google oauth credentials: %w", err)
	}

	payload := model.SetActiveGoogleOAuthCredentialsPayload{
		Credentials: &credentials,
	}

	return &payload, nil
}
