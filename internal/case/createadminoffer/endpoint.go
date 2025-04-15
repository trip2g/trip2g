package createadminoffer

import (
	"context"
	"net/http"
	"trip2g/internal/appreq"

	easyjson "github.com/mailru/easyjson"
)

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	var request Request
	if err := easyjson.Unmarshal(req.Req.PostBody(), &request); err != nil {
		return nil, err
	}
	return Resolve(context.Background(), req.Env.(Env), request)
}

func (Endpoint) Path() string {
	return "createadminoffer"
}

func (Endpoint) Method() string {
	return http.MethodPost
}
