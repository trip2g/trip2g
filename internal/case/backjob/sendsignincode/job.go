package sendsignincode

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_sign_in_code"
const QueueID = model.BackgroundDefaultQueue
const Priority = 0

type SendSignInCodeJob struct {
	enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *SendSignInCodeJob {
	return &SendSignInCodeJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t SendSignInCodeJob) EnqueueRequestSignInEmail(ctx context.Context, email string, code string) error {
	return t.enqueue(ctx, Params{Email: email, Code: code})
}
