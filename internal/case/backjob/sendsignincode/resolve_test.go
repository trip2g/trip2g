package sendsignincode

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type signInCodeCaptureLogger struct {
	lines []string
}

func (l *signInCodeCaptureLogger) record(msg string, keysAndValues ...interface{}) {
	l.lines = append(l.lines, fmt.Sprint(msg, keysAndValues))
}

func (l *signInCodeCaptureLogger) Info(msg string, keysAndValues ...interface{}) {
	l.record(msg, keysAndValues...)
}

func (l *signInCodeCaptureLogger) Error(msg string, keysAndValues ...interface{}) {
	l.record(msg, keysAndValues...)
}

func (l *signInCodeCaptureLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.record(msg, keysAndValues...)
}

func (l *signInCodeCaptureLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.record(msg, keysAndValues...)
}

type signInCodeTestEnv struct {
	log  *signInCodeCaptureLogger
	mail []model.Mail
}

func (e *signInCodeTestEnv) Logger() logger.Logger {
	return e.log
}

func (e *signInCodeTestEnv) SendMail(_ context.Context, mail model.Mail) error {
	e.mail = append(e.mail, mail)
	return nil
}

func TestResolveDoesNotLogSignInCode(t *testing.T) {
	const code = "otp-secret-927451"
	env := &signInCodeTestEnv{log: &signInCodeCaptureLogger{}}

	err := Resolve(context.Background(), env, Params{
		Email: "reader@example.com",
		Code:  code,
	})

	require.NoError(t, err)
	require.Len(t, env.mail, 1, "the successful SMTP path must be exercised")
	require.Contains(t, string(env.mail[0].Plain), code, "the code must still be delivered by email")
	require.NotContains(t, strings.Join(env.log.lines, "\n"), code,
		"one-time sign-in codes are credentials and must not be written to application logs")
}
