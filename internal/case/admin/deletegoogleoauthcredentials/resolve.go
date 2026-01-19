package deletegoogleoauthcredentials

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DeleteGoogleOAuthCredentials(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.DeleteGoogleOAuthCredentialsInput
type Payload = model.DeleteGoogleOAuthCredentialsOrErrorPayload

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

	err = env.DeleteGoogleOAuthCredentials(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete google oauth credentials: %w", err)
	}

	payload := model.DeleteGoogleOAuthCredentialsPayload{
		DeletedID: input.ID,
	}

	return &payload, nil
}
