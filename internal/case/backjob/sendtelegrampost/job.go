package sendtelegrampost

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_message"
const QueueID = model.BackgroundTelegramAPICallQueue
const Priority = 1 // shoud process before updates

type SendTelegramPostJob struct {
	enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *SendTelegramPostJob {
	return &SendTelegramPostJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t SendTelegramPostJob) EnqueueSendTelegramPost(ctx context.Context, params model.TelegramSendPostParams) error {
	return t.enqueue(ctx, params)
}
