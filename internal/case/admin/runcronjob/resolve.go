package runcronjob

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"trip2g/internal/cronjobs"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	CronJobs() *cronjobs.CronJobs
}

type Input = model.RunCronJobInput
type Payload = model.RunCronJobOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Name, ozzo.Required, is.PrintableASCII),
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

	// Get cron jobs manager
	cronJobsManager := env.CronJobs()
	if cronJobsManager == nil {
		return &model.ErrorPayload{Message: "Cron jobs manager not available"}, nil
	}

	// Manually trigger the job
	err = cronJobsManager.ExecuteJobManually(input.Name)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Failed to run cron job: %v", err)}, nil
	}

	payload := &model.RunCronJobPayload{
		Success: true,
	}

	return payload, nil
}