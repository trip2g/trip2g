package deleteboostycredentials

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	SoftDeleteBoostyCredentials(ctx context.Context, arg db.SoftDeleteBoostyCredentialsParams) (db.BoostyCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StopBoostyRefreshBackgroundJob(ctx context.Context, credentialsID int64) error
}

// Input is an alias for DeleteBoostyCredentialsInput for cleaner code.
type Input = model.DeleteBoostyCredentialsInput

// Payload is an alias for DeleteBoostyCredentialsOrErrorPayload for cleaner code.
type Payload = model.DeleteBoostyCredentialsOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	// Define params as separate variable for cleaner code
	params := db.SoftDeleteBoostyCredentialsParams{
		DeletedBy: ptr.To(int64(token.ID)),
		ID:        input.ID,
	}

	// Execute database operation
	_, err = env.SoftDeleteBoostyCredentials(ctx, params)
	if err != nil {
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to delete boosty credentials: %w", err)
	}

	err = env.StopBoostyRefreshBackgroundJob(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to stop boosty refresh job: %w", err)
	}

	// Define payload as separate variable
	payload := model.DeleteBoostyCredentialsPayload{
		DeletedID: input.ID,
	}

	return &payload, nil
}
