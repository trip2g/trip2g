package downloadonboardingvault

import (
	"fmt"
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

	if !token.IsAdmin() {
		ctx.SetStatusCode(http.StatusUnauthorized)
		return nil, nil
	}

	zipData, err := Resolve(ctx, env, token.ID)
	if err != nil {
		return nil, err
	}

	filename := makeFilename(env.PublicURL())

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("application/zip")
	ctx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	ctx.SetBody(zipData)

	return nil, nil
}

func (*Endpoint) Path() string {
	return "/_system/onboarding-vault"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}

// makeFilename creates a zip filename from the public URL domain.
// Example: "https://trip2g.com" -> "trip2g.com.zip".
func makeFilename(publicURL string) string {
	return domainFromURL(publicURL) + ".zip"
}
