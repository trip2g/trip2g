package main

import (
	"context"
	"fmt"
	"os"

	"trip2g/internal/db"
	"trip2g/internal/logger"
)

// cliEnv implements tgtd.ClientEnv for standalone use.
// Access hash cache methods are no-ops: a cache miss causes the client
// to do a live Telegram API lookup, which is fine for a read-only CLI.
type cliEnv struct{}

func (e *cliEnv) Logger() logger.Logger   { return &cliLogger{} }
func (e *cliEnv) LogLevel() string        { return "" }
func (e *cliEnv) DecryptData(b []byte) ([]byte, error) { return b, nil }

func (e *cliEnv) GetTelegramPublishAccountChatAccessHash(_ context.Context, _ db.GetTelegramPublishAccountChatAccessHashParams) (*string, error) {
	return nil, nil
}
func (e *cliEnv) GetTelegramPublishAccountInstantChatAccessHash(_ context.Context, _ db.GetTelegramPublishAccountInstantChatAccessHashParams) (*string, error) {
	return nil, nil
}
func (e *cliEnv) UpdateTelegramPublishAccountChatAccessHash(_ context.Context, _ db.UpdateTelegramPublishAccountChatAccessHashParams) error {
	return nil
}
func (e *cliEnv) UpdateTelegramPublishAccountInstantChatAccessHash(_ context.Context, _ db.UpdateTelegramPublishAccountInstantChatAccessHashParams) error {
	return nil
}

// cliLogger prints warnings/errors to stderr; discards info/debug.
type cliLogger struct{}

func (l *cliLogger) Info(_ string, _ ...interface{})  {}
func (l *cliLogger) Debug(_ string, _ ...interface{}) {}
func (l *cliLogger) Warn(msg string, kv ...interface{}) {
	fmt.Fprintf(os.Stderr, "warn: %s %v\n", msg, kv)
}
func (l *cliLogger) Error(msg string, kv ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: %s %v\n", msg, kv)
}
