package importtelegramchannel

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "import_telegram_channel"
const QueueID = model.BackgroundTelegramJobQueue
const Priority = 10 // lower priority than regular posts

type ImportTelegramChannelJob struct {
	enqueue jobs.EnqueueFunc
}

type ImportTelegramChannelEnv interface {
	jobs.Env
	Env
}

func New(env ImportTelegramChannelEnv) *ImportTelegramChannelJob {
	return &ImportTelegramChannelJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t ImportTelegramChannelJob) EnqueueImportTelegramChannel(ctx context.Context, params model.ImportTelegramChannelParams) error {
	return t.enqueue(ctx, params)
}
