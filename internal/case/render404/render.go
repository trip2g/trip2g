package render404

import (
	"trip2g/internal/appreq"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=.

type Params struct {
	UserToken *usertoken.Data
}

type Env interface {
	renderlayout.Env

	TrackNotFound(path string, ip string)
}

func Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req

	env, ok := req.Env.(Env)
	if !ok {
		return nil, appreq.ErrInvalidEnv
	}

	env.TrackNotFound(string(req.Req.Path()), req.Req.RemoteIP().String())

	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	defaulttemplate.WriteNotFound(ctx, env, token)
	return nil, nil
}
