package cleanupapikeylogs

import (
	"context"
)

// Job implements the cronjobs.Job interface.
type Job struct {
	env    Env
	config Config
}

func New(env Env, config Config) *Job {
	return &Job{env: env, config: config}
}

func (j *Job) Name() string {
	return "cleanup_api_key_logs"
}

// Schedule runs daily at 1:00 AM.
func (j *Job) Schedule() string {
	return "0 0 1 * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context) (any, error) {
	return Resolve(ctx, j.env, j.config)
}
