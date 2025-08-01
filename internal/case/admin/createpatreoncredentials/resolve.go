package createpatreoncredentials

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/case/refreshpatreondata"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertPatreonCredentials(ctx context.Context, arg db.InsertPatreonCredentialsParams) (db.PatreonCredential, error)
	UpsertPatreonCampaign(ctx context.Context, arg db.UpsertPatreonCampaignParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StartPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64, immediately bool) error

	refreshpatreondata.Env
}

// Input is an alias for CreatePatreonCredentialsInput for cleaner code.
type Input = model.CreatePatreonCredentialsInput

// Payload is an alias for CreatePatreonCredentialsOrErrorPayload for cleaner code.
type Payload = model.CreatePatreonCredentialsOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.CreatorAccessToken, ozzo.Required, ozzo.Length(10, 500)),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Always validate input first
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil // User-visible errors go in ErrorPayload
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	// Define params as separate variable for cleaner code
	params := db.InsertPatreonCredentialsParams{
		CreatedBy:          int64(token.ID),
		CreatorAccessToken: input.CreatorAccessToken,
	}

	// Execute database operation
	credentials, err := env.InsertPatreonCredentials(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "Patreon credentials already exist"}, nil
		}
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to insert patreon credentials: %w", err)
	}

	err = refreshpatreondata.Resolve(ctx, env, &credentials.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh Patreon data: %w", err)
	}

	err = env.StartPatreonRefreshBackgroundJob(ctx, credentials.ID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to start Patreon refresh background jobs: %w", err)
	}

	// Define payload as separate variable
	payload := model.CreatePatreonCredentialsPayload{
		PatreonCredentials: &credentials,
	}

	return &payload, nil
}
