package vacuumdatabase

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
	return "vacuum_database"
}

func (j *job) Schedule() string {
	// Run weekly on Sunday at 3 AM
	return "0 0 3 * * 0"
}

func (j *job) ExecuteAfterStart() bool {
	return false // Don't run immediately on startup
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env, Filter{})
}
