package refreshchartdata

import (
	"context"

	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "refresh_chart_data"
const QueueID = model.BackgroundDefaultQueue
const Priority = 0

type Job struct {
	enqueue jobs.EnqueueFunc
}

type JobEnv interface {
	jobs.Env
	Env
}

func New(env JobEnv) *Job {
	return &Job{
		enqueue: jobs.Register(env, QueueID, JobID, Priority,
			func(ctx context.Context, p Params) error {
				return Resolve(ctx, env, p)
			}),
	}
}

func (j *Job) EnqueueRefreshChartData(ctx context.Context, p Params) error {
	return j.enqueue(ctx, p)
}
