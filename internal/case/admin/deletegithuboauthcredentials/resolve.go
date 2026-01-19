package deletegithuboauthcredentials

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DeleteGitHubOAuthCredentials(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.DeleteGitHubOAuthCredentialsInput
type Payload = model.DeleteGitHubOAuthCredentialsOrErrorPayload

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

	err = env.DeleteGitHubOAuthCredentials(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete github oauth credentials: %w", err)
	}

	payload := model.DeleteGitHubOAuthCredentialsPayload{
		DeletedID: input.ID,
	}

	return &payload, nil
}
