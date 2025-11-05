package updatetelegrampost

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

const JobID = "update_message"
const QueueID = model.BackgroundTelegramAPICallQueue
const Priority = 0

type UpdateTelegramPostEnv interface {
	Env
	Logger() logger.Logger
	RegisterJob(qID model.BackgroundQueueID, id string, handler func(ctx context.Context, m []byte) error)
	EnqueueJob(ctx context.Context, job model.BackgroundTask) error
}

type UpdateTelegramPostJob struct {
	enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *UpdateTelegramPostJob {
	return &UpdateTelegramPostJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t UpdateTelegramPostJob) EnqueueUpdateTelegramPost(ctx context.Context, params model.TelegramUpdatePostParams) error {
	return t.enqueue(ctx, params)
}
