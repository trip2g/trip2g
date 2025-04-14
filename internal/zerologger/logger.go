package zerologger

import (
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"

	"trip2g/internal/logger"
)

type zerologLogger struct {
	logger zerolog.Logger
}

// Info ...
func (zl *zerologLogger) Info(msg string, keysAndValues ...interface{}) {
	zl.logger.Info().Fields(convertToFields(keysAndValues)).Msg(msg)
}

// Error ...
func (zl *zerologLogger) Error(msg string, keysAndValues ...interface{}) {
	zl.logger.Error().Fields(convertToFields(keysAndValues)).Msg(msg)
}

// Debug ...
func (zl *zerologLogger) Debug(msg string, keysAndValues ...interface{}) {
	zl.logger.Debug().Fields(convertToFields(keysAndValues)).Msg(msg)
}

// Warn ...
func (zl *zerologLogger) Warn(msg string, keysAndValues ...interface{}) {
	zl.logger.Warn().Fields(convertToFields(keysAndValues)).Msg(msg)
}

// convertToFields converts key-value pairs into a map.
func convertToFields(kv []interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			key, ok := kv[i].(string)
			if !ok {
				key = fmt.Sprintf("%v (non-string key)", kv[i])
			}

			m[key] = kv[i+1]
		}
	}
	return m
}

func New(logLevel string, prettyLogging bool) logger.Logger {
	var consoleWriter io.Writer

	if prettyLogging {
		consoleWriter = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		}
	} else {
		consoleWriter = os.Stderr
	}

	multi := zerolog.MultiLevelWriter(consoleWriter)

	zr := zerolog.New(multi).With().Timestamp().Logger()

	switch logLevel {
	case "debug":
		zr = zr.Level(zerolog.DebugLevel)
	case "info":
		zr = zr.Level(zerolog.InfoLevel)
	case "warn":
		zr = zr.Level(zerolog.WarnLevel)
	case "error":
		zr = zr.Level(zerolog.ErrorLevel)
	default:
		zr = zr.Level(zerolog.InfoLevel)
	}

	return &zerologLogger{
		logger: zr,
	}
}
