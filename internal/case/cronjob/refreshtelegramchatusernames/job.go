package refreshtelegramchatusernames

import (
	"context"
	"trip2g/internal/cronjobs"
)

const refreshBatchSize = 100

type Env interface {
	RefreshStaleTelegramChatUsernames(ctx context.Context, limit int) (int, error)
}

// Job delegates to a single app method; the logic stays in Execute rather than a
// resolve.go — a thin wrapper would add ceremony without testable behavior.
type job struct {
	env Env
}

func New(env Env) cronjobs.Job {
	return &job{env: env}
}

func (j *job) Name() string {
	return "refresh_telegram_chat_usernames"
}

func (j *job) Schedule() string {
	return "0 0 */6 * * *" // every 6 hours
}

func (j *job) ExecuteAfterStart() bool {
	return false
}

func (j *job) Execute(ctx context.Context) (any, error) {
	n, err := j.env.RefreshStaleTelegramChatUsernames(ctx, refreshBatchSize)
	if err != nil {
		return nil, err
	}
	return n, nil
}
