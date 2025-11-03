package updatetelegrampublishposts

import (
	"context"
)

type Job struct {
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

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // will checked in cmd/server/cronjobs.go
}
