package extractallnotionpages

import (
	"context"
)

type Job struct {
}

func (j *Job) Name() string {
	return "extract_all_notion_pages"
}

func (j *Job) Schedule() string {
	return "0 0 3 * * *" // every day at 3 AM
}

func (j *Job) ExecuteAfterStart() bool {
	return false // Don't run immediately on startup
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env), Params{}) //nolint:errcheck // will checked in cmd/server/cronjobs.go
}
