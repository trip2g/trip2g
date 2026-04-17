package refreshtelegramchatusernames

import "context"

const refreshBatchSize = 100

type Job struct{}

func (j *Job) Name() string {
	return "refresh_telegram_chat_usernames"
}

func (j *Job) Schedule() string {
	return "0 0 */6 * * *" // every 6 hours
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

type Env interface {
	RefreshStaleTelegramChatUsernames(ctx context.Context, limit int) (int, error)
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	n, err := env.(Env).RefreshStaleTelegramChatUsernames(ctx, refreshBatchSize) //nolint:errcheck // err is checked below
	if err != nil {
		return nil, err
	}
	return n, nil
}
