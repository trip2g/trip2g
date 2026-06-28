package expirestalewebhookdeliveries

import "context"

// Job implements the cronjobs.Job interface.
type Job struct{}

func (j *Job) Name() string { return "expire_stale_webhook_deliveries" }

// Schedule runs every minute to bound orphan-lock lifetime.
func (j *Job) Schedule() string { return "0 * * * * *" }

func (j *Job) ExecuteAfterStart() bool { return false }

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // checked in cmd/server/cronjobs.go.
}
