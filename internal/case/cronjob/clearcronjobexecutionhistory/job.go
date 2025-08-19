package clearcronjobexecutionhistory

import (
	"context"
)

type Job struct {
}

func (j *Job) Name() string {
	return "clear_cronjob_execution_history"
}

func (j *Job) Schedule() string {
	return "0 0 0 * * *" // every day at midnight
}

func (j *Job) ExecuteAfterStart() bool {
	return false // Don't run immediately on startup
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env), Filter{}) //nolint:errcheck // will checked in cmd/server/cronjobs.go
}
