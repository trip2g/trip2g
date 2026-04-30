package cleanupwebhookdeliverylogs

import (
	"context"
)

// Job implements the cronjobs.Job interface.
type Job struct{}

func (j *Job) Name() string {
	return "cleanup_webhook_delivery_logs"
}

// Schedule runs daily at midnight. Cron webhook response bodies store full agent
// payloads (hundreds of KB each), so logs must be pruned frequently to prevent
// the database from growing into the hundreds of MB range.
func (j *Job) Schedule() string {
	return "0 0 0 * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // checked in cmd/server/cronjobs.go.
}
