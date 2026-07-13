package regeneratenoteembeddings

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
	return "regenerate_note_embeddings"
}

func (j *job) Schedule() string {
	return "0 0 3 * * 0" // weekly on Sunday at 3:00 AM
}

func (j *job) ExecuteAfterStart() bool {
	return true // Run on startup to catch any missing embeddings
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
