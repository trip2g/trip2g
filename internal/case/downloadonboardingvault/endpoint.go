package downloadonboardingvault

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req

	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	if token == nil {
		ctx.SetStatusCode(http.StatusUnauthorized)
		return nil, nil
	}

	zipData, err := Resolve(ctx, env, token.ID)
	if err != nil {
		return nil, err
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("application/zip")
	ctx.Response.Header.Set("Content-Disposition", "attachment; filename=\"trip2g-vault.zip\"")
	ctx.SetBody(zipData)

	return nil, nil
}

func (*Endpoint) Path() string {
	return "/_system/onboarding-vault"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
