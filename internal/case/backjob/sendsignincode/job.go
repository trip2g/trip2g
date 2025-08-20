package sendsignincode

import (
	"context"
	"trip2g/internal/jobs"
)

const ID = "backjobs:send_sign_in_code"

type SendSignInCodeEnv interface {
	Env

	jobs.Env
}

type SendSignInCodeJob struct {
	env SendSignInCodeEnv
}

func New(env SendSignInCodeEnv) *SendSignInCodeJob {
	task := SendSignInCodeJob{env: env}
	jobs.Register(env, ID, Resolve)
	return &task
}

func (t SendSignInCodeJob) QueueRequestSignInEmail(ctx context.Context, email string, code string) error {
	return t.env.EnqueueJob(ctx, ID, Params{Email: email, Code: code})
}
