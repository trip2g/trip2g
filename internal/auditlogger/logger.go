package auditlogger

import (
	"context"
	"encoding/json"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger
	InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) error
}

type Config struct {
	LogLevel string
}

const (
	logLevelDebug = 1
	logLevelInfo  = 2
	logLevelWarn  = 3
	logLevelError = 4
)

type Logger struct {
	env Env
	log logger.Logger

	logLevel int
}

var _ logger.Logger = (*Logger)(nil)

func New(env Env, config Config) *Logger {
	l := Logger{
		env: env,
		log: logger.WithPrefix(env.Logger(), "audit: "),
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

	return &l
}

func (l *Logger) write(level int, msg string, keysAndValues ...interface{}) {
	if level < l.logLevel {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := logger.ConvertToFields(keysAndValues)

	jsonBytes, err := json.Marshal(params)
	if err != nil {
		l.log.Error("failed to marshal parameters to JSON", "error", err, "msg", msg)
	}

	arg := db.InsertAuditLogParams{
		Level:   int64(level),
		Message: msg,
		Params:  string(jsonBytes),
	}

	err = l.env.InsertAuditLog(ctx, arg)
	if err != nil {
		l.log.Error("failed to insert audit log", "error", err, "msg", msg, "params", keysAndValues)
		return
	}
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
