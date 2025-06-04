package sendsignincode

import (
	"bytes"
	"context"
	"fmt"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=.

type Params struct {
	Email string
	Code  string
}

type Env interface {
	Logger() logger.Logger
	SendMail(ctx context.Context, data model.Mail) error
}

func Resolve(ctx context.Context, env Env, task Params) error {
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
}
