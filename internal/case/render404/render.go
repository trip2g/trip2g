package render404

import (
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=.

type Params struct {
	UserToken *usertoken.Data
}

func Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req
	ctx.SetStatusCode(http.StatusNotFound)

	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	layoutParams := renderlayout.Params{
		Title: "Page not found",
	}

	pageParams := Params{
		UserToken: token,
	}

	return renderlayout.Handle(req, layoutParams, func() {
		WriteContent(ctx, &pageParams)
	})
}
