package updatehtmlinjection

import (
	"context"
	"database/sql"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	UpdateHTMLInjection(ctx context.Context, arg db.UpdateHTMLInjectionParams) (db.HtmlInjection, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

// Input is an alias for the GraphQL input type.
type Input = model.UpdateHTMLInjectionInput

// Payload is an alias for the GraphQL payload type.
type Payload = model.UpdateHTMLInjectionOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.ID, validation.Required),
		validation.Field(&r.Description, validation.Required),
		validation.Field(&r.Position, validation.Min(0)),
		validation.Field(&r.Placement, validation.Required, validation.In("head", "body_end")),
		validation.Field(&r.Content, validation.Required),
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

	// Define params as separate variable for cleaner code
	params := db.UpdateHTMLInjectionParams{
		ID:          input.ID,
		Description: input.Description,
		Position:    int64(input.Position),
		Placement:   input.Placement,
		Content:     input.Content,
		ActiveFrom:  sql.NullTime{},
		ActiveTo:    sql.NullTime{},
	}

	// Handle optional dates
	if input.ActiveFrom != nil {
		params.ActiveFrom = sql.NullTime{
			Time:  *input.ActiveFrom,
			Valid: true,
		}
	}

	if input.ActiveTo != nil {
		params.ActiveTo = sql.NullTime{
			Time:  *input.ActiveTo,
			Valid: true,
		}
	}

	// Execute database operation
	injection, err := env.UpdateHTMLInjection(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update HTML injection: %w", err)
	}

	// Define payload as separate variable
	payload := model.UpdateHTMLInjectionPayload{
		HTMLInjection: &injection,
	}

	return &payload, nil
}
