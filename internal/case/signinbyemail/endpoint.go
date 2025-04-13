package signinbyemail

import (
	"net/http"
	"trip2g/internal/appreq"

	easyjson "github.com/mailru/easyjson"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	request := Request{}

	err := easyjson.Unmarshal(req.Req.PostBody(), &request)
	if err != nil {
		return nil, err
	}

	return Resolve(req.Req, req.Env.(Env), request)
}

func (*Endpoint) Path() string {
	return "signinbyemail"
}

func (*Endpoint) Method() string {
	return http.MethodPost
}
