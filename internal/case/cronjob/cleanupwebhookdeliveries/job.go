package cleanupwebhookdeliveries

import (
	"context"
	"trip2g/internal/cronjobs"
)

// Job implements the cronjobs.Job interface.
type job struct {
	env Env
}

func New(env Env) cronjobs.Job {
	return &job{env: env}
}

func (j *job) Name() string {
	return "cleanup_webhook_deliveries"
}

// Schedule runs daily at 3:00 AM.
func (j *job) Schedule() string {
	return "0 0 3 * * *"
}

func (j *job) ExecuteAfterStart() bool {
	return false
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
