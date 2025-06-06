package revokeusersubgraphaccess

import (
	"context"
	"net/http"
	"trip2g/internal/appreq"

	easyjson "github.com/mailru/easyjson"
)

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	var request Request

	err := easyjson.Unmarshal(req.Req.PostBody(), &request)
	if err != nil {
		return nil, err
	}

	userToken, err := req.UserToken()
	if err != nil {
		return nil, err
	}
	request.UserToken = userToken

	if validateErr := request.Validate(); validateErr != nil {
		return nil, validateErr
	}

	return Resolve(context.Background(), req.Env.(Env), request)
}

func (Endpoint) Path() string {
	return "/api/admin/revokeusersubgraphaccess"
}

func (Endpoint) Method() string {
	return http.MethodPost
}
