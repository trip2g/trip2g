package refreshtelegramaccounts

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
	return "refresh_telegram_accounts"
}

func (j *Job) Schedule() string {
	return "0 0 3 * * *" // daily at 3:00 AM
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
