package synccli

import (
	"net/http"
	"sync"

	"trip2g/internal/appreq"
)

// Endpoint is public: the CLI is MIT open source.
type Endpoint struct {
	once   sync.Once
	script *Script
	err    error
}

func (*Endpoint) Path() string   { return "/_system/sync-cli.mjs" }
func (*Endpoint) Method() string { return http.MethodGet }

func (e *Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req

	// The archive is fixed for the life of the binary, so read it once.
	e.once.Do(func() {
		e.script, e.err = Resolve(req.Env.(Env))
	})

	if e.err != nil {
		return nil, e.err
	}

	if e.script == nil {
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.SetBodyString("sync cli is not bundled in this build\n")
		return nil, nil
	}

	ctx.Response.Header.Set("ETag", e.script.ETag)
	ctx.Response.Header.Set("Cache-Control", "no-cache")

	if string(ctx.Request.Header.Peek("If-None-Match")) == e.script.ETag {
		ctx.SetStatusCode(http.StatusNotModified)
		return nil, nil
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("text/javascript; charset=utf-8")
	ctx.SetBody(e.script.Body)

	return nil, nil
}
