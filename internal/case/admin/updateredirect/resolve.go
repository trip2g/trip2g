package updateredirect

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	UpdateRedirect(ctx context.Context, params db.UpdateRedirectParams) (db.Redirect, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func normalizeInput(i *model.UpdateRedirectInput) {
	i.Pattern = strings.TrimSpace(i.Pattern)
	i.Target = strings.TrimSpace(i.Target)
}

func validateInput(i *model.UpdateRedirectInput) *model.ErrorPayload {
	err := ozzo.ValidateStruct(i,
		ozzo.Field(&i.ID, ozzo.Required, ozzo.Min(1)),
		ozzo.Field(&i.Pattern, ozzo.Required),
		ozzo.Field(&i.Target, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err)
	}

	// Custom validation: if isRegex is true, pattern must be valid regex
	if i.IsRegex {
		_, compileErr := regexp.Compile(i.Pattern)
		if compileErr != nil {
			return &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "pattern", Value: "must be a valid regular expression"},
				},
			}
		}
	}

	return nil
}

type Input = model.UpdateRedirectInput
type Payload = model.UpdateRedirectOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	normalizeInput(&input)

	errorPayload := validateInput(&input)
	if errorPayload != nil {
		return errorPayload, nil
	}

	params := db.UpdateRedirectParams{
		ID:         input.ID,
		Pattern:    input.Pattern,
		IgnoreCase: input.IgnoreCase,
		IsRegex:    input.IsRegex,
		Target:     input.Target,
	}

	redirect, err := env.UpdateRedirect(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update redirect: %w", err)
	}

	response := model.UpdateRedirectPayload{
		Redirect: &redirect,
	}

	return &response, nil
}
