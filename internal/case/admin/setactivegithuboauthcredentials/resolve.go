package setactivegithuboauthcredentials

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	SetActiveGitHubOAuthCredentials(ctx context.Context, id int64) error
	GetGitHubOAuthCredentials(ctx context.Context, id int64) (db.GithubOauthCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.SetActiveGitHubOAuthCredentialsInput
type Payload = model.SetActiveGitHubOAuthCredentialsOrErrorPayload

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

	err = env.SetActiveGitHubOAuthCredentials(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to set active github oauth credentials: %w", err)
	}

	credentials, err := env.GetGitHubOAuthCredentials(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get github oauth credentials: %w", err)
	}

	payload := model.SetActiveGitHubOAuthCredentialsPayload{
		Credentials: &credentials,
	}

	return &payload, nil
}
