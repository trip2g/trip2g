package updateboostycredentials

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	UpdateBoostyCredentials(ctx context.Context, arg db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error)
}

// Input is an alias for UpdateBoostyCredentialsInput for cleaner code.
type Input = model.UpdateBoostyCredentialsInput

// Payload is an alias for UpdateBoostyCredentialsOrErrorPayload for cleaner code.
type Payload = model.UpdateBoostyCredentialsOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.ID, ozzo.Required),
		ozzo.Field(&r.AuthData, ozzo.When(r.AuthData != nil, ozzo.Length(10, 10000))),
		ozzo.Field(&r.DeviceID, ozzo.When(r.DeviceID != nil, ozzo.Length(5, 100))),
		ozzo.Field(&r.BlogName, ozzo.When(r.BlogName != nil, ozzo.Length(1, 100))),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Always validate input first
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil // User-visible errors go in ErrorPayload
	}

	// Check that at least one field is being updated
	if input.AuthData == nil && input.DeviceID == nil && input.BlogName == nil {
		return &model.ErrorPayload{Message: "No fields to update"}, nil
	}

	// Define params as separate variable for cleaner code
	params := db.UpdateBoostyCredentialsParams{
		ID:       input.ID,
		AuthData: sql.NullString{Valid: false},
		DeviceID: sql.NullString{Valid: false},
		BlogName: sql.NullString{Valid: false},
	}

	// Set fields only if provided
	if input.AuthData != nil {
		params.AuthData = sql.NullString{String: *input.AuthData, Valid: true}
	}
	if input.DeviceID != nil {
		params.DeviceID = sql.NullString{String: *input.DeviceID, Valid: true}
	}
	if input.BlogName != nil {
		params.BlogName = sql.NullString{String: *input.BlogName, Valid: true}
	}

	// Execute database operation
	credentials, err := env.UpdateBoostyCredentials(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Boosty credentials not found"}, nil
		}
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to update boosty credentials: %w", err)
	}

	// Define payload as separate variable
	payload := model.UpdateBoostyCredentialsPayload{
		BoostyCredentials: &credentials,
	}

	return &payload, nil
}
