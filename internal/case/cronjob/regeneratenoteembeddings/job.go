package regeneratenoteembeddings

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
	return "regenerate_note_embeddings"
}

func (j *Job) Schedule() string {
	return "0 0 3 * * 0" // weekly on Sunday at 3:00 AM
}

func (j *Job) ExecuteAfterStart() bool {
	return true // Run on startup to catch any missing embeddings
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
