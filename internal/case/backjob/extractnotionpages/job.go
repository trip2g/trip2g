package extractnotionpages

import (
	"context"
	"trip2g/internal/jobs"
)

const ID = "backjobs:extract_notion_pages"

type ExtractNotionPagesEnv interface {
	Env

	jobs.Env
}

type ExtractNotionPagesJob struct {
	env ExtractNotionPagesEnv
}

func New(env ExtractNotionPagesEnv) *ExtractNotionPagesJob {
	task := ExtractNotionPagesJob{env: env}
	jobs.Register(env, ID, Resolve)
	return &task
}

func (t ExtractNotionPagesJob) QueueExtractNotionPages(ctx context.Context, pageID *string) error {
	return t.env.EnqueueJob(ctx, ID, Params{PageID: pageID})
}
