package sendsignincode

import (
	"context"
	"time"
	"trip2g/internal/logger"

	"github.com/mikestefanello/backlite"
)

type Task struct {
	Email string
	Code  int64
}

type Env interface {
	Logger() logger.Logger
}

func (t Task) Config() backlite.QueueConfig {
	return backlite.QueueConfig{
		Name:        "SendSignInCodeTask",
		MaxAttempts: 3,
		Timeout:     5 * time.Second,
		Backoff:     10 * time.Second,
		Retention: &backlite.Retention{
			Duration:   24 * time.Hour,
			OnlyFailed: false,
			Data: &backlite.RetainData{
				OnlyFailed: false,
			},
		},
	}
}

func NewQueue(env Env) backlite.Queue {
	return backlite.NewQueue[Task](func(ctx context.Context, task Task) error {
		env.Logger().Info("Sending sign-in code to %s: %d", task.Email, task.Code)

		time.Sleep(5 * time.Second) // Simulate sending the code

		return nil
	})
}
