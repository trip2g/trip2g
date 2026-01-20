package regeneratenoteembeddings

import (
	"context"
)

type Job struct{}

func (j *Job) Name() string {
	return "regenerate_note_embeddings"
}

func (j *Job) Schedule() string {
	return "0 0 3 * * *" // daily at 3:00 AM
}

func (j *Job) ExecuteAfterStart() bool {
	return true // Run on startup to catch any missing embeddings
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // error is returned
}
