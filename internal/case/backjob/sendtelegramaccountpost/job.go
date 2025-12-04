package sendtelegramaccountpost

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_telegram_account_post"
const QueueID = model.BackgroundTelegramJobQueue
const Priority = 1 // should process before updates

type SendTelegramAccountPostJob struct {
	enqueue jobs.EnqueueFunc
}

type SendTelegramAccountPostEnv interface {
	jobs.Env
	Env
}

func New(env SendTelegramAccountPostEnv) *SendTelegramAccountPostJob {
	return &SendTelegramAccountPostJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t SendTelegramAccountPostJob) EnqueueSendTelegramAccountPost(ctx context.Context, params model.SendTelegramPublishPostParams) error {
	return t.enqueue(ctx, params)
}
