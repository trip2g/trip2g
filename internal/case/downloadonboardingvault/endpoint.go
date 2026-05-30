package downloadonboardingvault

import (
	"fmt"
	"net/http"
	"trip2g/internal/appreq"
	onboardingvault "trip2g/onboarding-vault"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req

	if len(onboardingvault.ZipData) == 0 {
		ctx.SetStatusCode(http.StatusNotFound)
		return nil, nil
	}

	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	if token == nil || !token.IsAdmin() {
		ctx.SetStatusCode(http.StatusUnauthorized)
		return nil, nil
	}

	// ?enable_admin_graphql issues the API key with MCP admin tools enabled,
	// so the vault's agent can run admin GraphQL without a separate toggle.
	// Bare presence (?enable_admin_graphql) enables it; an explicit false value
	// (=false / =0) opts out.
	qa := ctx.QueryArgs()
	enableAdminGraphQL := qa.Has("enable_admin_graphql") &&
		(len(qa.Peek("enable_admin_graphql")) == 0 || qa.GetBool("enable_admin_graphql"))

	zipData, err := Resolve(ctx, env, token.ID, enableAdminGraphQL)
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
