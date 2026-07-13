package updatetelegrampublishposts

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
	return "update_telegram_publish_posts"
}

func (j *job) Schedule() string {
	return "0 0 0 * * *" // daily at midnight
}

func (j *job) ExecuteAfterStart() bool {
	return false
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
