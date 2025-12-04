package sendtelegramaccountmessage

import (
	"context"
	"trip2g/internal/case/backjob/updatetelegramaccountmessage"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_account_message"
const QueueID = model.BackgroundTelegramAPICallQueue
const Priority = updatetelegramaccountmessage.Priority + 1

type SendTelegramAccountMessageJob struct {
	enqueue jobs.EnqueueFunc
}

type SendTelegramAccountMessageEnv interface {
	jobs.Env
	Env
}

func New(env SendTelegramAccountMessageEnv) *SendTelegramAccountMessageJob {
	return &SendTelegramAccountMessageJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t SendTelegramAccountMessageJob) EnqueueSendTelegramAccountMessage(ctx context.Context, params model.TelegramAccountSendPostParams) error {
	return t.enqueue(ctx, params)
}
