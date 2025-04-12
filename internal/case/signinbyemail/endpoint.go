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

	response, err := Resolve(req.Req, req.Env.(Env), request)
	if err != nil {
		return nil, err
	}

	// admins can signout from the user session later.
	if response.tokenData != nil {
		token, storeErr := req.TokenManager.Store(req.Req, *response.tokenData)
		if storeErr != nil {
			return nil, storeErr
		}

		response.Token = token
	} else {
		delErr := req.TokenManager.Delete(req.Req)
		if delErr != nil {
			return nil, delErr
		}
	}

	return response, nil
}

func (*Endpoint) Path() string {
	return "signinbyemail"
}

func (*Endpoint) Method() string {
	return http.MethodPost
}
