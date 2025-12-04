package refreshtelegramappconfig

import (
	"context"
)

type Job struct{}

func (j *Job) Name() string {
	return "refresh_telegram_app_config"
}

func (j *Job) Schedule() string {
	return "0 0 3 * * *" // daily at 3:00 AM
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env))
}
