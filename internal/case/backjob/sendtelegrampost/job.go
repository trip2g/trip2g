package sendtelegrampost

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_telegram_post"
const QueueID = model.BackgroundTelegramJobQueue
const Priority = 1 // shoud process before updates

type SendTelegramPostJob struct {
	enqueue jobs.EnqueueFunc
}

type SendTelegramPostEnv interface {
	jobs.Env
	Env
}

func New(env SendTelegramPostEnv) *SendTelegramPostJob {
	return &SendTelegramPostJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t SendTelegramPostJob) EnqueueSendTelegramPost(ctx context.Context, params model.SendTelegramPublishPostParams) error {
	return t.enqueue(ctx, params)
}
