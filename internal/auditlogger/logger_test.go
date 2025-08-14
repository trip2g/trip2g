package auditlogger

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type mockEnv struct {
	insertedLogs []db.InsertAuditLogParams
}

func (m *mockEnv) Logger() logger.Logger {
	return &logger.TestLogger{Prefix: "[TEST]"}
}

func (m *mockEnv) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) error {
	m.insertedLogs = append(m.insertedLogs, arg)
	return nil
}

func TestAuditLogger(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		logLevel       string
		logFunc        func(*Logger)
		shouldLog      bool
		expectedLevel  int64
		expectedParams map[string]interface{}
	}{
		{
			name:     "info log with params",
			config:   Config{LogLevel: "info"},
			logLevel: "info",
			logFunc: func(l *Logger) {
				l.Info("test message", "key1", "value1", "key2", 123)
			},
			shouldLog:      true,
			expectedLevel:  logLevelInfo,
			expectedParams: map[string]interface{}{"key1": "value1", "key2": float64(123)},
		},
		{
			name:     "debug log when level is info - should not log",
			config:   Config{LogLevel: "info"},
			logLevel: "debug",
			logFunc: func(l *Logger) {
				l.Debug("debug message", "key", "value")
			},
			shouldLog: false,
		},
		{
			name:     "error log",
			config:   Config{LogLevel: "error"},
			logLevel: "error",
			logFunc: func(l *Logger) {
				l.Error("error occurred", "error", "test error", "code", 500)
			},
			shouldLog:      true,
			expectedLevel:  logLevelError,
			expectedParams: map[string]interface{}{"error": "test error", "code": float64(500)},
		},
		{
			name:     "warn log",
			config:   Config{LogLevel: "warn"},
			logLevel: "warn",
			logFunc: func(l *Logger) {
				l.Warn("warning message", "threshold", 0.8, "current", 0.9)
			},
			shouldLog:      true,
			expectedLevel:  logLevelWarn,
			expectedParams: map[string]interface{}{"threshold": 0.8, "current": 0.9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &mockEnv{}
			logger := New(env, tt.config)

			tt.logFunc(logger)

			if tt.shouldLog {
				require.Len(t, env.insertedLogs, 1)
				log := env.insertedLogs[0]
				require.Equal(t, tt.expectedLevel, log.Level)

				// Parse params JSON
				if tt.expectedParams != nil {
					var params map[string]interface{}
					err := json.Unmarshal([]byte(log.Params), &params)
					require.NoError(t, err)
					require.Equal(t, tt.expectedParams, params)
				}
			} else {
				require.Empty(t, env.insertedLogs)
			}
		})
	}
}

func TestAuditLoggerInvalidLogLevel(t *testing.T) {
	env := &mockEnv{}
	require.Panics(t, func() {
		New(env, Config{LogLevel: "invalid"})
	})
}
