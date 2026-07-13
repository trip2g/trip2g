package materializegitmirror

import (
	"context"
)

type Job struct {
	env Env
}

func New(env Env) *Job {
	return &Job{env: env}
}

func (j *Job) Name() string {
	return "materialize_git_mirror"
}

func (j *Job) Schedule() string {
	return "0 0 0 * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false // Don't run immediately on startup
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
