package createredirect

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
	InsertRedirect(ctx context.Context, params db.InsertRedirectParams) (db.Redirect, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func normalizeInput(i *model.CreateRedirectInput) {
	i.Pattern = strings.TrimSpace(i.Pattern)
	i.Target = strings.TrimSpace(i.Target)
}

func validateInput(i *model.CreateRedirectInput) *model.ErrorPayload {
	err := ozzo.ValidateStruct(i,
		ozzo.Field(&i.Pattern, ozzo.Required),
		ozzo.Field(&i.Target, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err)
	}

	// Custom validation: if isRegex is true, pattern must be valid regex
	if i.IsRegex {
		_, err := regexp.Compile(i.Pattern)
		if err != nil {
			return &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "pattern", Value: "must be a valid regular expression"},
				},
			}
		}
	}

	return nil
}

func Resolve(ctx context.Context, env Env, input model.CreateRedirectInput) (model.CreateRedirectOrErrorPayload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	normalizeInput(&input)

	errorPayload := validateInput(&input)
	if errorPayload != nil {
		return errorPayload, nil
	}

	params := db.InsertRedirectParams{
		CreatedBy:  int64(token.ID),
		Pattern:    input.Pattern,
		IgnoreCase: input.IgnoreCase,
		IsRegex:    input.IsRegex,
		Target:     input.Target,
	}

	redirect, err := env.InsertRedirect(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create redirect: %w", err)
	}

	response := model.CreateRedirectPayload{
		Redirect: &redirect,
	}

	return &response, nil
}