package getadminpage

import (
	"fmt"
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	path := req.Req.QueryArgs().Peek("path")
	if path == nil {
		return nil, &appreq.Error{
			Code:    http.StatusBadRequest,
			Message: "path is required",
		}
	}

	fmt.Println("path:", string(path))

	return Resolve(req.Req, req.Env.(Env), Request{
		Path: string(path),
	})
}

func (*Endpoint) Path() string {
	return "getadminpage"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
