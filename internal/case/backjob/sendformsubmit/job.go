package sendformsubmit

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_form_submit_email"
const QueueID = model.BackgroundDefaultQueue
const Priority = 0

type SendFormSubmitEmailJob struct {
	enqueue jobs.EnqueueFunc
}

type SendFormSubmitEmailEnv interface {
	jobs.Env
	Env
}

func New(env SendFormSubmitEmailEnv) *SendFormSubmitEmailJob {
	return &SendFormSubmitEmailJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority,
			func(ctx context.Context, params Params) error {
				return Resolve(ctx, env, params)
			}),
	}
}

func (t SendFormSubmitEmailJob) EnqueueSendFormSubmit(ctx context.Context, submitID int64) error {
	return t.enqueue(ctx, Params{SubmitID: submitID})
}
