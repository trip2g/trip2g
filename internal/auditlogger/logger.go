package auditlogger

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger
	InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) error
}

type Config struct {
	LogLevel     string
	FlushTimeout time.Duration
}

const (
	logLevelDebug = 1
	logLevelInfo  = 2
	logLevelWarn  = 3
	logLevelError = 4
)

type logEntry struct {
	level   int
	message string
	params  string
}

type Logger struct {
	env Env
	log logger.Logger

	logLevel     int
	flushTimeout time.Duration

	buffer   []logEntry
	bufferMu sync.Mutex
	ticker   *time.Ticker
}

var _ logger.Logger = (*Logger)(nil)

func New(ctx context.Context, env Env, config Config) *Logger {
	flushTimeout := config.FlushTimeout
	if flushTimeout == 0 {
		flushTimeout = 5 * time.Second
	}

	l := Logger{
		env:          env,
		log:          logger.WithPrefix(env.Logger(), "audit:"),
		flushTimeout: flushTimeout,
	}

	switch config.LogLevel {
	case "debug":
		l.logLevel = logLevelDebug
	case "info":
		l.logLevel = logLevelInfo
	case "warn":
		l.logLevel = logLevelWarn
	case "error":
		l.logLevel = logLevelError
	default:
		panic("invalid log level: " + config.LogLevel)
	}

	l.ticker = time.NewTicker(flushTimeout)
	go l.flushLoop(ctx)

	return &l
}

func (l *Logger) write(level int, msg string, keysAndValues ...interface{}) {
	if level < l.logLevel {
		return
	}

	params := logger.ConvertToFields(keysAndValues)

	jsonBytes, err := json.Marshal(params)
	if err != nil {
		l.log.Error("failed to marshal parameters to JSON", "error", err, "msg", msg)
		return
	}

	entry := logEntry{
		level:   level,
		message: msg,
		params:  string(jsonBytes),
	}

	l.bufferMu.Lock()
	l.buffer = append(l.buffer, entry)
	l.bufferMu.Unlock()
}

func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.log.Info(msg, keysAndValues...)
	l.write(logLevelInfo, msg, keysAndValues...)
}

func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.log.Error(msg, keysAndValues...)
	l.write(logLevelError, msg, keysAndValues...)
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.log.Debug(msg, keysAndValues...)
	l.write(logLevelDebug, msg, keysAndValues...)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.log.Warn(msg, keysAndValues...)
	l.write(logLevelWarn, msg, keysAndValues...)
}

func (l *Logger) flushLoop(ctx context.Context) {
	for {
		select {
		case <-l.ticker.C:
			l.flush()
		case <-ctx.Done():
			l.ticker.Stop()
			l.flush()
			return
		}
	}
}

func (l *Logger) flush() {
	l.bufferMu.Lock()
	if len(l.buffer) == 0 {
		l.bufferMu.Unlock()
		return
	}

	entries := make([]logEntry, len(l.buffer))
	copy(entries, l.buffer)
	l.bufferMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sent := 0
	for i, entry := range entries {
		arg := db.InsertAuditLogParams{
			Level:   int64(entry.level),
			Message: entry.message,
			Params:  entry.params,
		}

		err := l.env.InsertAuditLog(ctx, arg)
		if err != nil {
			if db.IsSQLiteBusyError(err) {
				l.log.Warn("SQLite busy error during audit log flush", "error", err, "sent", sent, "total", len(entries))
				l.returnToBuffer(entries[i:])
				return
			}

			l.log.Error("failed to insert audit log during flush", "error", err, "entry", entry)
		}
		sent++
	}

	l.bufferMu.Lock()
	if len(l.buffer) >= len(entries) {
		l.buffer = l.buffer[len(entries):]
	} else {
		l.buffer = l.buffer[:0]
	}
	l.bufferMu.Unlock()

	if sent > 0 {
		l.log.Debug("audit logs flushed", "count", sent)
	}
}

func (l *Logger) returnToBuffer(failedEntries []logEntry) {
	l.bufferMu.Lock()
	defer l.bufferMu.Unlock()

	newBuffer := make([]logEntry, 0, len(failedEntries)+len(l.buffer))
	newBuffer = append(newBuffer, failedEntries...)
	newBuffer = append(newBuffer, l.buffer...)
	l.buffer = newBuffer
}
