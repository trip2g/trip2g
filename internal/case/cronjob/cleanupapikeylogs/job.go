package cleanupapikeylogs

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
	return "cleanup_api_key_logs"
}

// Schedule runs daily at 1:00 AM.
func (j *job) Schedule() string {
	return "0 0 1 * * *"
}

func (j *job) ExecuteAfterStart() bool {
	return false
}

func (j *job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env, j.config)
}
