package updatetelegramaccountpost

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "update_telegram_account_post"
const QueueID = model.BackgroundTelegramAPICallQueue
const Priority = 0

type UpdateTelegramAccountPostJob struct {
	enqueue jobs.EnqueueFunc
}

type UpdateTelegramAccountPostEnv interface {
	jobs.Env
	Env
}

func New(env UpdateTelegramAccountPostEnv) *UpdateTelegramAccountPostJob {
	return &UpdateTelegramAccountPostJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t UpdateTelegramAccountPostJob) EnqueueUpdateTelegramAccountPost(ctx context.Context, notePathID int64) error {
	return t.enqueue(ctx, notePathID)
}
