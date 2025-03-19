package logger

import "fmt"

// Logger is the interface that wraps the basic logging methods.
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
}

// TestLogger for tests.
type TestLogger struct {
	Prefix string
}

// Info ...
func (l *TestLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Println(l.Prefix, msg, keysAndValues) //nolint:forbidigo // just for tests
}

// Error ...
func (l *TestLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Println(l.Prefix, msg, keysAndValues) //nolint:forbidigo // just for tests
}

// Debug ...
func (l *TestLogger) Debug(msg string, keysAndValues ...interface{}) {
	fmt.Println(l.Prefix, msg, keysAndValues) //nolint:forbidigo // just for tests
}

// Warn ...
func (l *TestLogger) Warn(msg string, keysAndValues ...interface{}) {
	fmt.Println(l.Prefix, msg, keysAndValues) //nolint:forbidigo // just for tests
}
