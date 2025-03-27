package logger

const space = " "

type withPrefix struct {
	logger Logger
	prefix string
}

func WithPrefix(logger Logger, prefix string) Logger {
	return &withPrefix{logger, prefix + space}
}

// Info ...
func (l *withPrefix) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(l.prefix+msg, keysAndValues...)
}

// Error ...
func (l *withPrefix) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error(l.prefix+msg, keysAndValues...)
}

// Debug ...
func (l *withPrefix) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(l.prefix+msg, keysAndValues...)
}

// Warn ...
func (l *withPrefix) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(l.prefix+msg, keysAndValues...)
}
