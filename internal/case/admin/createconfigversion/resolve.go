//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg createconfigversion_test . Env

package createconfigversion

import (
	"context"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertConfigVersion(ctx context.Context, params db.InsertConfigVersionParams) (db.ConfigVersion, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.CreateConfigVersionInput
type Payload = model.CreateConfigVersionOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.Timezone, validation.Required, validation.By(validateTimezone)),
	))
}

func validateTimezone(value interface{}) error {
	timezone, ok := value.(string)
	if !ok {
		return fmt.Errorf("timezone must be a string")
	}

	_, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	return nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	params := db.InsertConfigVersionParams{
		CreatedBy:         int64(token.ID),
		ShowDraftVersions: input.ShowDraftVersions,
		DefaultLayout:     input.DefaultLayout,
		Timezone:          input.Timezone,
	}

	createdConfigVersion, err := env.InsertConfigVersion(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert config version: %w", err)
	}

	payload := model.CreateConfigVersionPayload{
		ConfigVersion: &createdConfigVersion,
	}

	return &payload, nil
}