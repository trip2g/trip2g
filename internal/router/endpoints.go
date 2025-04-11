package router

import (
	"trip2g/internal/case/getnotehashes"
	"trip2g/internal/case/listadminnotepaths"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/logger"
)

// to add new endpoints do two things:
// 1. Add new endpoints here.
//
//nolint:gochecknoglobals // readonly routes
var endpoints = []Endpoint{
	&getnotehashes.Endpoint{},
	&listadminnotepaths.Endpoint{},
	&pushnotes.Endpoint{},
}

// 2. the endpoint env interfaces here.
type Env interface {
	Logger() logger.Logger

	getnotehashes.Env
	listadminnotepaths.Env
	pushnotes.Env
}
