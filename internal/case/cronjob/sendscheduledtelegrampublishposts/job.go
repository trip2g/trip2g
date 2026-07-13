package sendscheduledtelegrampublishposts

import (
	"context"
	"trip2g/internal/cronjobs"
)

type job struct {
	env Env
	// cron is the schedule expression, injected from appconfig (env/flag).
	cron string
}

func New(env Env, cron string) cronjobs.Job {
	return &job{env: env, cron: cron}
}

func (j *job) Name() string {
	return "send_scheduled_telegram_publishposts"
}

func (j *job) Schedule() string {
	if j.cron != "" {
		return j.cron
	}
	return "0 * * * * *" // default: every minute
}

func (j *job) ExecuteAfterStart() bool {
	return true // Don't run immediately on startup
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
