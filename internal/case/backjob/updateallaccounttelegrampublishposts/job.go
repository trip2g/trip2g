package updateallaccounttelegrampublishposts

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "update_all_account_telegram_publish_posts"
const QueueID = model.BackgroundTelegramJobQueue
const Priority = 0

type UpdateAllAccountTelegramPublishPostsJob struct {
	enqueue jobs.EnqueueFunc
}

type UpdateAllAccountTelegramPublishPostsEnv interface {
	jobs.Env
	Env
}

func New(env UpdateAllAccountTelegramPublishPostsEnv) *UpdateAllAccountTelegramPublishPostsJob {
	return &UpdateAllAccountTelegramPublishPostsJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority,
			func(ctx context.Context, params Params) error {
				return Resolve(ctx, env, params)
			}),
	}
}

func (t UpdateAllAccountTelegramPublishPostsJob) EnqueueUpdateAllAccountTelegramPublishPosts(ctx context.Context, accountID int64) error {
	return t.enqueue(ctx, Params{AccountID: accountID})
}
