package sendformsubmit

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"trip2g/internal/logger"
	"trip2g/internal/model"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=.

type FieldEntry struct {
	Name  string
	Value string
}

type Params struct {
	SubmitID    int64
	NotePath    string
	SubmittedAt string
	UserInfo    string
	IP          string
	Fields      []FieldEntry
	AdminURL    string
}

type Env interface {
	Logger() logger.Logger
	SendMail(ctx context.Context, data model.Mail) error
	GetAdminEmails(ctx context.Context) ([]string, error)
	GetFormSubmitForEmail(ctx context.Context, submitID int64) (*Params, error)
}

func Resolve(ctx context.Context, env Env, params Params) error {
	full, err := env.GetFormSubmitForEmail(ctx, params.SubmitID)
	if err != nil {
		return fmt.Errorf("sendformsubmit: get submit: %w", err)
	}
	if full == nil {
		env.Logger().Warn("form submit not found for email", "submit_id", params.SubmitID)
		return nil
	}

	emails, err := env.GetAdminEmails(ctx)
	if err != nil {
		return fmt.Errorf("sendformsubmit: get admin emails: %w", err)
	}

	var buf bytes.Buffer
	WritePlainView(&buf, full)
	body := buf.Bytes()

	subject := fmt.Sprintf("New form submission: %s | %s", full.NotePath, time.Now().Format("2006-01-02"))

	for _, email := range emails {
		if err := env.SendMail(ctx, model.Mail{
			To:      email,
			Subject: subject,
			Plain:   body,
		}); err != nil {
			env.Logger().Error("sendformsubmit: send mail failed", "email", email, "error", err)
		}
	}
	return nil
}
