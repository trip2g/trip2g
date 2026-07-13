package executecronwebhooks

import (
	"context"
)

// Job implements the cronjobs.Job interface.
type Job struct {
	env Env
	// cron is the schedule expression, injected from appconfig (env/flag).
	cron string
}

func New(env Env, cron string) *Job {
	return &Job{env: env, cron: cron}
}

func (j *Job) Name() string {
	return "execute_cron_webhooks"
}

// Schedule returns the configured cron expression; defaults to every minute.
func (j *Job) Schedule() string {
	if j.cron != "" {
		return j.cron
	}
	return "0 * * * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
