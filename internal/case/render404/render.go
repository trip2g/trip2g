package render404

import (
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/renderlayout"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=.

func Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req
	ctx.SetStatusCode(http.StatusNotFound)

	layoutParams := renderlayout.Params{
		Title: "Page not found",
	}

	return renderlayout.Handle(req, layoutParams, func() {
		WriteContent(ctx)
	})
}
