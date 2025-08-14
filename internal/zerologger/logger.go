package zerologger

import (
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
	zl.logger.Info().Fields(logger.ConvertToFields(keysAndValues)).Msg(msg)
}

// Error ...
func (zl *zerologLogger) Error(msg string, keysAndValues ...interface{}) {
	zl.logger.Error().Fields(logger.ConvertToFields(keysAndValues)).Msg(msg)
}

// Debug ...
func (zl *zerologLogger) Debug(msg string, keysAndValues ...interface{}) {
	zl.logger.Debug().Fields(logger.ConvertToFields(keysAndValues)).Msg(msg)
}

// Warn ...
func (zl *zerologLogger) Warn(msg string, keysAndValues ...interface{}) {
	zl.logger.Warn().Fields(logger.ConvertToFields(keysAndValues)).Msg(msg)
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
