package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
	"trip2g/internal/case/backjob/sendformsubmit"
	"trip2g/internal/case/backjob/sendsignincode"
	"trip2g/internal/case/requestemailsignin"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

func (a *app) SendMail(_ context.Context, data model.Mail) error {
	if a.config.SMTPHost == "" {
		a.log.Warn("send email skipped: no SMTP/RESEND configured", "to", data.To, "subject", data.Subject)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", a.config.SMTPHost, a.config.SMTPPort)

	var auth smtp.Auth
	if a.config.SMTPUser != "" {
		auth = smtp.PlainAuth("", a.config.SMTPUser, a.config.SMTPPass, a.config.SMTPHost)
	}

	body := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		data.To, a.config.MailFrom, data.Subject, string(data.Plain))

	if a.config.SMTPStartTLS { //nolint:nestif // SMTP send has inherent nested control flow
		c, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
		defer c.Close()

		err = c.StartTLS(&tls.Config{ServerName: a.config.SMTPHost, MinVersion: tls.VersionTLS12})
		if err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}

		if auth != nil {
			err = c.Auth(auth)
			if err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}

		err = c.Mail(a.config.MailFrom)
		if err != nil {
			return fmt.Errorf("smtp mail from: %w", err)
		}

		err = c.Rcpt(data.To)
		if err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}

		wc, err := c.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}

		_, err = fmt.Fprint(wc, body)
		if err != nil {
			return fmt.Errorf("smtp write: %w", err)
		}

		return wc.Close()
	}

	// Plaintext SMTP: sends an explicit conversation and never issues STARTTLS,
	// even if the server advertises it (smtp.SendMail does opportunistic
	// STARTTLS regardless of this flag, which breaks e.g. a local Postfix relay
	// presenting a self-signed cert for its own hostname).
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer c.Close()

	if auth != nil {
		err = c.Auth(auth)
		if err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	err = c.Mail(a.config.MailFrom)
	if err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	err = c.Rcpt(data.To)
	if err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	_, err = fmt.Fprint(wc, body)
	if err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	return wc.Close()
}

func (a *app) LogSignInCodes() bool {
	return a.config.LogSignInCodes
}

func (a *app) SMTPHost() string {
	return a.config.SMTPHost
}

func (a *app) EnqueueSendFormSubmitEmail(ctx context.Context, submitID int64) error {
	return a.SendFormSubmitEmailJob.EnqueueSendFormSubmit(ctx, submitID)
}

func (a *app) GetFormSubmitForEmail(ctx context.Context, submitID int64) (*sendformsubmit.Params, error) {
	submit, err := a.Queries.GetFormSubmitByID(ctx, submitID)
	if err != nil {
		if db.IsNoFound(err) {
			return nil, nil
		}
		return nil, err
	}

	notes := a.LatestNoteViews()
	notePath := ""
	if note := notes.GetByVersionID(submit.NoteVersionID); note != nil {
		notePath = note.Path
	}
	userInfo := "guest"
	if submit.UserID != nil {
		userInfo = fmt.Sprintf("user:%d", *submit.UserID)
	}

	strs, _ := a.Queries.GetFormStringValuesBySubmitID(ctx, submitID)
	ints, _ := a.Queries.GetFormIntValuesBySubmitID(ctx, submitID)
	bools, _ := a.Queries.GetFormBoolValuesBySubmitID(ctx, submitID)

	var fields []sendformsubmit.FieldEntry
	for _, s := range strs {
		fields = append(fields, sendformsubmit.FieldEntry{Name: s.FieldName, Value: s.Value})
	}
	for _, n := range ints {
		fields = append(fields, sendformsubmit.FieldEntry{Name: n.FieldName, Value: strconv.FormatInt(n.Value, 10)})
	}
	for _, b := range bools {
		boolVal := "false"
		if b.Value != 0 {
			boolVal = "true"
		}
		fields = append(fields, sendformsubmit.FieldEntry{Name: b.FieldName, Value: boolVal})
	}

	return &sendformsubmit.Params{
		SubmitID:    submitID,
		NotePath:    notePath,
		SubmittedAt: submit.CreatedAt.Format("2006-01-02 15:04:05"),
		UserInfo:    userInfo,
		IP:          submit.Ip,
		Fields:      fields,
	}, nil
}

var _ sendformsubmit.Env = (*app)(nil)
var _ sendsignincode.Env = (*app)(nil)
var _ requestemailsignin.Env = (*app)(nil)
