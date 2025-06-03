package sendsignincode

import (
	"bytes"
	"context"
	"fmt"
	"time"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/mikestefanello/backlite"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=.

type Task struct {
	Email string
	Code  string
}

type Env interface {
	Logger() logger.Logger
	SendMail(ctx context.Context, data model.Mail) error
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

		var buf bytes.Buffer
		WritePlainView(&buf, task)

		data := model.Mail{
			To:      task.Email,
			Subject: "Sign-in Code | trip2g",
			Plain:   buf.Bytes(),
		}

		err := env.SendMail(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to send sign-in code: %w", err)
		}

		return nil
	})
}
