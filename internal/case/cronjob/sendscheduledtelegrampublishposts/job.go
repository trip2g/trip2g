package sendscheduledtelegrampublishposts

import (
	"context"
)

type Job struct {
	// Cron is the schedule expression, injected from appconfig (env/flag).
	Cron string
}

func (j *Job) Name() string {
	return "send_scheduled_telegram_publishposts"
}

func (j *Job) Schedule() string {
	if j.Cron != "" {
		return j.Cron
	}
	return "0 * * * * *" // default: every minute
}

func (j *Job) ExecuteAfterStart() bool {
	return true // Don't run immediately on startup
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // will checked in cmd/server/cronjobs.go
}
