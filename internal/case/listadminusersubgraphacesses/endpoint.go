package listadminusersubgraphacesses

import (
	"context"
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	var request Request

	if err := request.Validate(); err != nil {
		return nil, err
	}

	return Resolve(context.Background(), req.Env.(Env), request)
}

func (Endpoint) Path() string {
	return "/api/listadminusersubgraphacesses"
}

func (Endpoint) Method() string {
	return http.MethodGet
}
