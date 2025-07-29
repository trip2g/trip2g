package createboostycredentials

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertBoostyCredentials(ctx context.Context, arg db.InsertBoostyCredentialsParams) (db.BoostyCredential, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

// Input is an alias for CreateBoostyCredentialsInput for cleaner code.
type Input = model.CreateBoostyCredentialsInput

// Payload is an alias for CreateBoostyCredentialsOrErrorPayload for cleaner code.
type Payload = model.CreateBoostyCredentialsOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.AuthData, ozzo.Required, ozzo.Length(10, 10000)),
		ozzo.Field(&r.DeviceID, ozzo.Required, ozzo.Length(5, 100)),
		ozzo.Field(&r.BlogName, ozzo.Required, ozzo.Length(1, 100)),
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
	params := db.InsertBoostyCredentialsParams{
		CreatedBy: int64(token.ID),
		AuthData:  input.AuthData,
		DeviceID:  input.DeviceID,
		BlogName:  input.BlogName,
	}

	// Execute database operation
	credentials, err := env.InsertBoostyCredentials(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "Boosty credentials already exist"}, nil
		}
		// System errors are returned as error (will show generic message to user)
		return nil, fmt.Errorf("failed to insert boosty credentials: %w", err)
	}

	// Define payload as separate variable
	payload := model.CreateBoostyCredentialsPayload{
		BoostyCredentials: &credentials,
	}

	return &payload, nil
}
