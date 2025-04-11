package router

import "trip2g/internal/case/getnotehashes"

// to add new endpoints do two things:
// 1. Add new endpoints here.
var endpoints = []Endpoint{
	&getnotehashes.Endpoint{},
}

// 2. the endpoint env interfaces here.
type Env interface {
	getnotehashes.Env
}
