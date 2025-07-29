package deleteboostycredentials

import (
	"context"
	"database/sql"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	SoftDeleteBoostyCredentials(ctx context.Context, arg db.SoftDeleteBoostyCredentialsParams) (db.BoostyCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
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
		DeletedBy: sql.NullInt64{Int64: int64(token.ID), Valid: true},
		ID:        input.ID,
	}

	// Execute database operation
	_, err = env.SoftDeleteBoostyCredentials(ctx, params)
	if err != nil {
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to delete boosty credentials: %w", err)
	}

	// Define payload as separate variable
	payload := model.DeleteBoostyCredentialsPayload{
		DeletedID: input.ID,
	}

	return &payload, nil
}
