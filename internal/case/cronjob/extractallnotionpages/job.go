package extractallnotionpages

import (
	"context"
	"trip2g/internal/case/backjob/extractnotionpages"
)

type Env = extractnotionpages.Env

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
	params := extractnotionpages.Params{PageID: nil}

	//nolint:errcheck // env is guaranteed to be of type extractnotionpages.Env
	err := extractnotionpages.Resolve(ctx, env.(extractnotionpages.Env), params)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
