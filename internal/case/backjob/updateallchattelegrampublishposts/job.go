package updateallchattelegrampublishposts

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "update_all_chat_telegram_publish_posts"
const QueueID = model.BackgroundTelegramJobQueue
const Priority = 0

type UpdateAllChatTelegramPublishPostsJob struct {
	enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *UpdateAllChatTelegramPublishPostsJob {
	return &UpdateAllChatTelegramPublishPostsJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t UpdateAllChatTelegramPublishPostsJob) EnqueueUpdateAllChatTelegramPublishPosts(ctx context.Context, chatID int64) error {
	return t.enqueue(ctx, Params{ChatID: chatID})
}
