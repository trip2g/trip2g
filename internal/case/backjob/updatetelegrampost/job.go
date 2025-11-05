package updatetelegrampost

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "update_telegram_post"
const QueueID = model.BackgroundTelegramJobQueue
const Priority = 0

type UpdateTelegramPostJob struct {
	enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *UpdateTelegramPostJob {
	return &UpdateTelegramPostJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t UpdateTelegramPostJob) EnqueueUpdateTelegramPost(ctx context.Context, notePathID int64) error {
	return t.enqueue(ctx, notePathID)
}
