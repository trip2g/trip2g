package router

import (
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/logger"
)

type Env interface {
	RoutesEnv

	renderlayout.Env

	Logger() logger.Logger
}
