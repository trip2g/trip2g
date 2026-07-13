package removeexpiredtgchatmembers

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
	return "remove_expired_tg_chat_members"
}

func (j *Job) Schedule() string {
	return "0 0 * * * *" // every hour
}

func (j *Job) ExecuteAfterStart() bool {
	return true
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env, Filter{})
}
