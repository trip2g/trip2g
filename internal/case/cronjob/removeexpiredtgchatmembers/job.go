package removeexpiredtgchatmembers

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
	return "remove_expired_tg_chat_members"
}

func (j *job) Schedule() string {
	return "0 0 * * * *" // every hour
}

func (j *job) ExecuteAfterStart() bool {
	return true
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env, Filter{})
}
