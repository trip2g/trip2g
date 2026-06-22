package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// spyLogger records the level each message was logged at.
type spyLogger struct {
	levels map[string]string // msg -> level
}

func newSpyLogger() *spyLogger { return &spyLogger{levels: map[string]string{}} }

func (l *spyLogger) Info(msg string, _ ...interface{})  { l.levels[msg] = "info" }
func (l *spyLogger) Error(msg string, _ ...interface{}) { l.levels[msg] = "error" }
func (l *spyLogger) Debug(msg string, _ ...interface{}) { l.levels[msg] = "debug" }
func (l *spyLogger) Warn(msg string, _ ...interface{})  { l.levels[msg] = "warn" }

func TestRunnerLoggerDowngradesRoutineInfoToDebug(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		// Routine per-job spam — should be downgraded to debug.
		{"running job", "Running job", "debug"},
		{"ran job", "Ran job", "debug"},
		{"extending timeout", "Extending message timeout", "debug"},
		{"starting runner", "Starting job runner", "debug"},
		{"stopping runner", "Stopping job runner", "debug"},
		{"stopped runner", "Stopped job runner", "debug"},
		// Failures must stay visible at info.
		{"error running job", "Error running job", "info"},
		{"error receiving job", "Error receiving job", "info"},
		{"recovered from panic", "Recovered from panic in job", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := newSpyLogger()
			rl := runnerLogger{spy}

			rl.Info(tt.msg, "name", "x", "id", "y")

			require.Equal(t, tt.want, spy.levels[tt.msg])
		})
	}
}
