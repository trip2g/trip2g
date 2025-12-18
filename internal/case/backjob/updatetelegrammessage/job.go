package updatetelegrammessage

import (
	"context"
	"trip2g/internal/case/backjob/updatetelegrampost"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "update_message"
const QueueID = model.BackgroundTelegramBotAPIQueue
const Priority = updatetelegrampost.Priority + 1

type UpdateTelegramMessageJob struct {
	enqueue jobs.EnqueueFunc
}

type UpdateTelegramMessageEnv interface {
	jobs.Env
	Env
}

func New(env UpdateTelegramMessageEnv) *UpdateTelegramMessageJob {
	return &UpdateTelegramMessageJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t UpdateTelegramMessageJob) EnqueueUpdateTelegramMessage(ctx context.Context, params model.TelegramUpdatePostParams) error {
	return t.enqueue(ctx, params)
}
