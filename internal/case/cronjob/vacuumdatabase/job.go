package vacuumdatabase

import (
	"context"
)

type Job struct {
}

func (j *Job) Name() string {
	return "vacuum_database"
}

func (j *Job) Schedule() string {
	// Run weekly on Sunday at 3 AM
	return "0 0 3 * * 0"
}

func (j *Job) ExecuteAfterStart() bool {
	return false // Don't run immediately on startup
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	//nolint:errcheck // env is guaranteed to be of type vacuumdatabase.Env
	return Resolve(ctx, env.(Env), Filter{})
}
