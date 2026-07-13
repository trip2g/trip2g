package sendscheduledtelegrampublishposts

import (
	"context"
)

type Job struct {
	env Env
	// cron is the schedule expression, injected from appconfig (env/flag).
	cron string
}

func New(env Env, cron string) *Job {
	return &Job{env: env, cron: cron}
}

func (j *Job) Name() string {
	return "send_scheduled_telegram_publishposts"
}

func (j *Job) Schedule() string {
	if j.cron != "" {
		return j.cron
	}
	return "0 * * * * *" // default: every minute
}

func (j *Job) ExecuteAfterStart() bool {
	return true // Don't run immediately on startup
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
