package deletepatreoncredentials

import (
	"context"
	"database/sql"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	SoftDeletePatreonCredentials(ctx context.Context, arg db.SoftDeletePatreonCredentialsParams) (db.PatreonCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StopPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64) error
}

// Input is an alias for DeletePatreonCredentialsInput for cleaner code.
type Input = model.DeletePatreonCredentialsInput

// Payload is an alias for DeletePatreonCredentialsOrErrorPayload for cleaner code.
type Payload = model.DeletePatreonCredentialsOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	// Define params as separate variable for cleaner code
	params := db.SoftDeletePatreonCredentialsParams{
		DeletedBy: sql.NullInt64{Int64: int64(token.ID), Valid: true},
		ID:        input.ID,
	}

	// Execute database operation
	_, err = env.SoftDeletePatreonCredentials(ctx, params)
	if err != nil {
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to delete patreon credentials: %w", err)
	}

	err = env.StopPatreonRefreshBackgroundJob(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to stop patreon refresh background jobs: %w", err)
	}

	// Define payload as separate variable
	payload := model.DeletePatreonCredentialsPayload{
		DeletedID: input.ID,
	}

	return &payload, nil
}
