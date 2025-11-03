package updateallchattelegrampublishposts

import (
	"context"
	"trip2g/internal/jobs"
)

const ID = "backjobs:update_all_chat_telegram_publish_posts"

type UpdateAllChatTelegramPublishPostsEnv interface {
	Env

	jobs.Env
}

type UpdateAllChatTelegramPublishPostsJob struct {
	env UpdateAllChatTelegramPublishPostsEnv
}

func New(env UpdateAllChatTelegramPublishPostsEnv) *UpdateAllChatTelegramPublishPostsJob {
	task := UpdateAllChatTelegramPublishPostsJob{env: env}
	jobs.Register(env, ID, Resolve)
	return &task
}

func (t UpdateAllChatTelegramPublishPostsJob) QueueUpdateAllChatTelegramPublishPosts(ctx context.Context, chatID int64) error {
	return t.env.EnqueueJob(ctx, ID, Params{ChatID: chatID})
}
