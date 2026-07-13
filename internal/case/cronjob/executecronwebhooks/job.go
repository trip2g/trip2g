package executecronwebhooks

import (
	"context"
	"trip2g/internal/cronjobs"
)

// Job implements the cronjobs.Job interface.
type job struct {
	env Env
	// cron is the schedule expression, injected from appconfig (env/flag).
	cron string
}

func New(env Env, cron string) cronjobs.Job {
	return &job{env: env, cron: cron}
}

func (j *job) Name() string {
	return "execute_cron_webhooks"
}

// Schedule returns the configured cron expression; defaults to every minute.
func (j *job) Schedule() string {
	if j.cron != "" {
		return j.cron
	}
	return "0 * * * * *"
}

func (j *job) ExecuteAfterStart() bool {
	return false
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
