package tgtd

import "trip2g/internal/logger"

// Env defines external dependencies for the tgtd package.
type Env interface {
	Logger() logger.Logger
}
