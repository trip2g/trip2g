package restorepatreoncredentials

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	RestorePatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StartPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64, immediately bool) error
}

// Input is an alias for RestorePatreonCredentialsInput for cleaner code.
type Input = model.RestorePatreonCredentialsInput

// Payload is an alias for RestorePatreonCredentialsOrErrorPayload for cleaner code.
type Payload = model.RestorePatreonCredentialsOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	// Execute database operation
	credentials, err := env.RestorePatreonCredentials(ctx, input.ID)
	if err != nil {
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to restore patreon credentials: %w", err)
	}

	err = env.StartPatreonRefreshBackgroundJob(ctx, credentials.ID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to start Patreon refresh background jobs: %w", err)
	}

	// Define payload as separate variable
	payload := model.RestorePatreonCredentialsPayload{
		PatreonCredentials: &credentials,
	}

	return &payload, nil
}
