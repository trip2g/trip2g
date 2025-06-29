package createemailwaitlistrequest

import (
	"context"
	"database/sql"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	InsertWaitListEmailRequest(ctx context.Context, arg db.InsertWaitListEmailRequestParams) error
	RequestIP(ctx context.Context) string
}

type Input = model.CreateEmailWaitListRequestInput
type Payload = model.CreateEmailWaitListRequestOrErrorPayload

func validateInput(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.Email, validation.Required, is.Email),
		validation.Field(&r.PathID, validation.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateInput(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	ip := env.RequestIP(ctx)

	params := db.InsertWaitListEmailRequestParams{
		Email:      input.Email,
		NotePathID: input.PathID,
		Ip:         sql.NullString{String: ip, Valid: ip != ""},
	}

	err := env.InsertWaitListEmailRequest(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			// Email already exists in wait list, ignore duplicate
		} else {
			return nil, fmt.Errorf("failed to insert wait list email request: %w", err)
		}
	}

	payload := model.CreateEmailWaitListRequestPayload{
		Success: true,
	}

	return &payload, nil
}
