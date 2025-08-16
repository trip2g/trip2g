package removeexpiredtgchatmembers

import "context"

type Job struct {
}

func (j *Job) Name() string {
	return "remove_expired_tg_chat_members"
}

func (j *Job) Schedule() string {
	return "0 0 * * * *" // every hour
}

func (j *Job) ExecuteAfterStart() bool {
	return true
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env), Filter{})
}
