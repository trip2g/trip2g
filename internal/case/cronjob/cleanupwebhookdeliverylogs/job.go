package cleanupwebhookdeliverylogs

import (
	"context"
	"trip2g/internal/cronjobs"
)

// Job implements the cronjobs.Job interface.
type job struct {
	env    Env
	config Config
}

func New(env Env, config Config) cronjobs.Job {
	return &job{env: env, config: config}
}

func (j *job) Name() string {
	return "cleanup_webhook_delivery_logs"
}

// Schedule runs daily at midnight. Cron webhook response bodies store full agent
// payloads (hundreds of KB each), so a long retention window grows the database
// into the hundreds of MB range.
func (j *job) Schedule() string {
	return "0 0 0 * * *"
}

func (j *job) ExecuteAfterStart() bool {
	return false
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env, j.config)
}
