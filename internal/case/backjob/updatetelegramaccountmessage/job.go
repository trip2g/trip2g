package updatetelegramaccountmessage

import (
	"context"
	"trip2g/internal/case/backjob/updatetelegramaccountpost"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "update_account_message"
const QueueID = model.BackgroundTelegramAccountAPIQueue
const Priority = updatetelegramaccountpost.Priority + 1

type UpdateTelegramAccountMessageJob struct {
	enqueue jobs.EnqueueFunc
}

type UpdateTelegramAccountMessageEnv interface {
	jobs.Env
	Env
}

func New(env UpdateTelegramAccountMessageEnv) *UpdateTelegramAccountMessageJob {
	return &UpdateTelegramAccountMessageJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t UpdateTelegramAccountMessageJob) EnqueueUpdateTelegramAccountMessage(ctx context.Context, params model.TelegramAccountUpdatePostParams) error {
	return t.enqueue(ctx, params)
}
