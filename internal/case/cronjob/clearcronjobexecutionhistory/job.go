package clearcronjobexecutionhistory

import (
	"context"
	"trip2g/internal/cronjobs"
)

type job struct {
	env Env
}

func New(env Env) cronjobs.Job {
	return &job{env: env}
}

func (j *job) Name() string {
	return "clear_cronjob_execution_history"
}

func (j *job) Schedule() string {
	return "0 0 0 * * *" // every day at midnight
}

func (j *job) ExecuteAfterStart() bool {
	return false // Don't run immediately on startup
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env, Filter{})
}
