package auditlogger

import "trip2g/internal/logger"

type Env interface {
	Logger() logger.Logger
}

type Config struct {
	logLevel string
}

const (
	logLevelDebug = 1
	logLevelInfo  = 2
	logLevelWarn  = 3
	logLevelError = 4
)

type AuditLogger struct {
	env Env
	log logger.Logger

	logLevel int
}

var _ logger.Logger = (*AuditLogger)(nil)

func New(env Env, config Config) *AuditLogger {
	l := AuditLogger{
		env: env,
		log: logger.WithPrefix(env.Logger(), "audit: "),
	}

	switch config.logLevel {
	case "debug":
		l.logLevel = logLevelDebug
	case "info":
		l.logLevel = logLevelInfo
	case "warn":
		l.logLevel = logLevelWarn
	case "error":
		l.logLevel = logLevelError
	default:
		panic("invalid log level: " + config.logLevel)
	}

	return &l
}

func (l *AuditLogger) write(level int, msg string, keysAndValues ...interface{}) {
	if level < l.logLevel {
		return
	}
}

func (l *AuditLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log.Info(msg, keysAndValues...)
	l.write(logLevelInfo, msg, keysAndValues...)
}

func (l *AuditLogger) Error(msg string, keysAndValues ...interface{}) {
	l.log.Error(msg, keysAndValues...)
	l.write(logLevelError, msg, keysAndValues...)
}

func (l *AuditLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.log.Debug(msg, keysAndValues...)
	l.write(logLevelDebug, msg, keysAndValues...)
}

func (l *AuditLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.log.Warn(msg, keysAndValues...)
	l.write(logLevelWarn, msg, keysAndValues...)
}
