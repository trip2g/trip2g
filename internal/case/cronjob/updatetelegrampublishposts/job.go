package updatetelegrampublishposts

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
	return "update_telegram_publish_posts"
}

func (j *Job) Schedule() string {
	return "0 0 0 * * *" // daily at midnight
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
