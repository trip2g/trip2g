package downloadonboardingvault

import (
	"errors"
	"fmt"
	"net/http"
	"trip2g/internal/appreq"

	"github.com/valyala/fasthttp"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req

	if len(env.OnboardingVaultZip()) == 0 {
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

	vaultName, err := resolveVaultName(qa, env.PublicURL())
	if err != nil {
		var appErr *appreq.Error
		if errors.As(err, &appErr) {
			ctx.SetStatusCode(appErr.Code)
			ctx.SetBodyString(appErr.Message)
			return nil, nil
		}

		return nil, err
	}

	zipData, err := Resolve(ctx, env, token.ID, enableAdminGraphQL, vaultName)
	if err != nil {
		return nil, err
	}

	filename := vaultName + ".zip"

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

// resolveVaultName picks the archive name: explicit ?name wins, otherwise the
// instance domain. It names both "<name>.zip" and the "<name>/" folder inside,
// so a caller that passes it knows where to unpack from instead of guessing.
// A present but invalid ?name is an error — never a silent fallback.
func resolveVaultName(qa *fasthttp.Args, publicURL string) (string, error) {
	if !qa.Has("name") {
		return domainFromURL(publicURL), nil
	}

	name := string(qa.Peek("name"))

	err := validateVaultName(name)
	if err != nil {
		return "", err
	}

	return name, nil
}
