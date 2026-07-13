package expirestalewebhookdeliveries

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

func (j *job) Name() string { return "expire_stale_webhook_deliveries" }

// Schedule runs every minute to bound orphan-lock lifetime.
func (j *job) Schedule() string { return "0 * * * * *" }

func (j *job) ExecuteAfterStart() bool { return false }

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env)
}
