package generatenoteversionembedding

import (
	"context"

	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "generate_note_version_embedding"
const QueueID = model.BackgroundDefaultQueue
const Priority = 10 // Lower priority than user-facing jobs

type Job struct {
	enqueue jobs.EnqueueFunc
}

type JobEnv interface {
	jobs.Env
	Env
}

func New(env JobEnv) *Job {
	return &Job{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (j *Job) Enqueue(ctx context.Context, versionID int64) error {
	return j.enqueue(ctx, Params{VersionID: versionID})
}
