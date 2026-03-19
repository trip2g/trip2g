package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	tdclock "github.com/gotd/td/clock"

	"trip2g/internal/db"
	"trip2g/internal/logger"
)

// offsetClock shifts time.Now() by a fixed duration to correct system clock skew.
type offsetClock struct{ offset time.Duration }

func (c offsetClock) Now() time.Time                    { return time.Now().Add(c.offset) }
func (c offsetClock) Timer(d time.Duration) tdclock.Timer  { return sysTimer{time.NewTimer(d)} }
func (c offsetClock) Ticker(d time.Duration) tdclock.Ticker { return sysTicker{time.NewTicker(d)} }

type sysTimer struct{ t *time.Timer }

func (s sysTimer) C() <-chan time.Time   { return s.t.C }
func (s sysTimer) Reset(d time.Duration) { s.t.Reset(d) }
func (s sysTimer) Stop() bool            { return s.t.Stop() }

type sysTicker struct{ t *time.Ticker }

func (s sysTicker) C() <-chan time.Time   { return s.t.C }
func (s sysTicker) Stop()                 { s.t.Stop() }
func (s sysTicker) Reset(d time.Duration) { s.t.Reset(d) }

func buildLogger(debug bool) *zap.Logger {
	if !debug {
		return nil
	}
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	l, _ := cfg.Build()
	return l
}

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
