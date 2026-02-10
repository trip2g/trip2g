package cleanupwebhookdeliverylogs

import (
	"context"
)

// Job implements the cronjobs.Job interface.
type Job struct{}

func (j *Job) Name() string {
	return "cleanup_webhook_delivery_logs"
}

// Schedule runs every hour.
func (j *Job) Schedule() string {
	return "0 0 * * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // checked in cmd/server/cronjobs.go.
}
