package extractnotionpages

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "extract_notion_pages"
const QueueID = model.BackgroundDefaultQueue
const Priority = 0

type ExtractNotionPagesJob struct {
	enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *ExtractNotionPagesJob {
	return &ExtractNotionPagesJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t ExtractNotionPagesJob) EnqueueExtractNotionPages(ctx context.Context, pageID *string) error {
	return t.enqueue(ctx, Params{PageID: pageID})
}
