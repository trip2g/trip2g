package clearcronjobexecutionhistory

import (
	"context"
	"fmt"
)

type Env interface {
	DeleteOldCronJobExecutions(ctx context.Context) (int64, error)
}

type Filter struct {
	// No filters needed for this job
}

type Result struct {
	DeletedCount int64
}

func Resolve(ctx context.Context, env Env, filter Filter) (*Result, error) {
	deletedCount, err := env.DeleteOldCronJobExecutions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old cron job executions: %w", err)
	}

	return &Result{
		DeletedCount: deletedCount,
	}, nil
}
