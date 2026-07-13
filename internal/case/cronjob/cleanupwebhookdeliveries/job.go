package cleanupwebhookdeliveries

import (
	"context"
)

// Job implements the cronjobs.Job interface.
type Job struct {
	env Env
}

func New(env Env) *Job {
	return &Job{env: env}
}

func (j *Job) Name() string {
	return "cleanup_webhook_deliveries"
}

// Schedule runs daily at 3:00 AM.
func (j *Job) Schedule() string {
	return "0 0 3 * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
