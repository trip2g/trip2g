package cleanupwebhookdeliveries

import (
	"context"
)

// Job implements the cronjobs.Job interface.
type Job struct{}

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

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // checked in cmd/server/cronjobs.go.
}
