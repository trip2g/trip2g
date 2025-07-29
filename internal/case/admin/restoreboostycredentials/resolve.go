package restoreboostycredentials

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	RestoreBoostyCredentials(ctx context.Context, id int64) (db.BoostyCredential, error)
}

// Input is an alias for RestoreBoostyCredentialsInput for cleaner code.
type Input = model.RestoreBoostyCredentialsInput

// Payload is an alias for RestoreBoostyCredentialsOrErrorPayload for cleaner code.
type Payload = model.RestoreBoostyCredentialsOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Execute database operation
	credentials, err := env.RestoreBoostyCredentials(ctx, input.ID)
	if err != nil {
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to restore boosty credentials: %w", err)
	}

	// Define payload as separate variable
	payload := model.RestoreBoostyCredentialsPayload{
		BoostyCredentials: &credentials,
	}

	return &payload, nil
}
