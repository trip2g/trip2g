package runcronjob

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	ExecuteCronJobManually(jobID int64) (*db.CronJobExecution, error)
	Logger() logger.Logger
}

type Input = model.RunCronJobInput
type Payload = model.RunCronJobOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.ID, ozzo.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	logger := logger.WithPrefix(env.Logger(), "RunCronJob:")

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

	logger.Info("Running cron job manually", "id", input.ID)

	// Manually trigger the job
	execution, err := env.ExecuteCronJobManually(input.ID)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Failed to run cron job: %v", err)}, nil
	}

	payload := &model.RunCronJobPayload{
		Execution: execution,
	}

	return payload, nil
}
