package extractnotionpage

import (
	"context"
	"trip2g/internal/jobs"
)

const ID = "backjobs:extract_notion_page"

type ExtractNotionPageEnv interface {
	Env

	jobs.Env
}

type ExtractNotionPageJob struct {
	env ExtractNotionPageEnv
}

func New(env ExtractNotionPageEnv) *ExtractNotionPageJob {
	task := ExtractNotionPageJob{env: env}
	jobs.Register(env, ID, Resolve)
	return &task
}

func (t ExtractNotionPageJob) QueueExtractNotionPage(ctx context.Context, pageID string) error {
	return t.env.EnqueueJob(ctx, ID, Params{PageID: pageID})
}