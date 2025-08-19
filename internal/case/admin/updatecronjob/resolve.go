package updatecronjob

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UpdateCronJob(ctx context.Context, arg db.UpdateCronJobParams) (db.CronJob, error)
	CronJobByID(ctx context.Context, id int64) (db.CronJob, error)
}

type Input = model.UpdateCronJobInput
type Payload = model.UpdateCronJobOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.ID, ozzo.Required),
		ozzo.Field(&r.Expression, ozzo.Required, is.PrintableASCII),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Check admin authorization
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Validate input
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Update the cron job
	params := db.UpdateCronJobParams{
		ID:         input.ID,
		Enabled:    input.Enabled,
		Expression: input.Expression,
	}

	updatedJob, err := env.UpdateCronJob(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update cron job: %w", err)
	}

	// Convert to GraphQL model
	cronJob := &model.AdminCronJob{
		ID:         updatedJob.ID,
		Name:       updatedJob.Name,
		Enabled:    updatedJob.Enabled,
		Expression: updatedJob.Expression,
	}

	if updatedJob.LastExecAt.Valid {
		cronJob.LastExecAt = &updatedJob.LastExecAt.Time
	}

	payload := &model.UpdateCronJobPayload{
		CronJob: cronJob,
	}

	return payload, nil
}
