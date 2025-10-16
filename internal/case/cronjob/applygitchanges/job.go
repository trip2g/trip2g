package applygitchanges

import (
	"context"
)

type Job struct {
}

func (j *Job) Name() string {
	return "apply_git_changes"
}

func (j *Job) Schedule() string {
	return "0 0 0 * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false // Don't run immediately on startup
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // will checked in cmd/server/cronjobs.go
}
