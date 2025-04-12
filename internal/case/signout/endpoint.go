package signout

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	response, err := Resolve(req.Req, req.Env.(Env), Request{})
	if err != nil {
		return nil, err
	}

	// admins can signout from the user session later.
	if response.tokenData != nil {
		token, err := req.TokenManager.Store(req.Req, *response.tokenData)
		if err != nil {
			return nil, err
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
	return "signout"
}

func (*Endpoint) Method() string {
	return http.MethodPost
}
