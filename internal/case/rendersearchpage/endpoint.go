package rendersearchpage

import (
	"fmt"
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/case/sitesearch"
	"trip2g/internal/graph/model"
)

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=. -ext=html

type Endpoint struct{}

type Env = sitesearch.Env

type Response struct {
	Query  string
	Result *model.SearchConnection
}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	// token, err := req.UserToken()
	// if err != nil {
	// 	return nil, err
	// }

	request := model.SearchInput{
		Query: string(req.Req.QueryArgs().Peek("q")),
	}

	ctx := req.Req

	res, err := sitesearch.Resolve(ctx, req.Env.(sitesearch.Env), request)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve search: %w", err)
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	layoutParams := renderlayout.Params{
		Title: "Результаты поиска",
	}

	resp := Response{
		Result: res,
		Query:  request.Query,
	}

	return renderlayout.Handle(req, layoutParams, func() {
		WritePage(ctx, &resp)
	})
}

func (Endpoint) Path() string {
	return "/search"
}

func (Endpoint) Method() string {
	return http.MethodGet
}
