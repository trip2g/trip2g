package sendtelegrammessage

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_message"
const QueueID = model.BackgroundTelegramAPICallQueue
const Priority = 1 // shoud process before updates

type SendTelegramMessageJob struct {
	enqueue jobs.EnqueueFunc
}

type SendTelegramMessageEnv interface {
	jobs.Env
	Env
}

func New(env SendTelegramMessageEnv) *SendTelegramMessageJob {
	return &SendTelegramMessageJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t SendTelegramMessageJob) EnqueueSendTelegramMessage(ctx context.Context, params model.TelegramSendPostParams) error {
	return t.enqueue(ctx, params)
}
