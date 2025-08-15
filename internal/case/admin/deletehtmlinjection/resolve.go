package deletehtmlinjection

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DeleteHTMLInjection(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

// Input is an alias for the GraphQL input type.
type Input = model.DeleteHTMLInjectionInput

// Payload is an alias for the GraphQL payload type.
type Payload = model.DeleteHTMLInjectionOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.ID, validation.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Check admin authorization
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Always validate input first
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Execute database operation
	err = env.DeleteHTMLInjection(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete HTML injection: %w", err)
	}

	// Define payload as separate variable
	payload := model.DeleteHTMLInjectionPayload{
		DeletedID: input.ID,
	}

	return &payload, nil
}
