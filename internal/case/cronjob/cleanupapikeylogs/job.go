package cleanupapikeylogs

import (
	"context"
)

// Job implements the cronjobs.Job interface.
type Job struct {
	Config Config
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

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env), j.Config) //nolint:errcheck // checked in cmd/server/cronjobs.go.
}
