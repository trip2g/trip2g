package downloadonboardingvault

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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

var nonAlphanumRE = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// makeFilename creates a clean filename from the public URL.
// Example: "https://trip2g.com" -> "trip2g-vault.zip".
func makeFilename(publicURL string) string {
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.Host == "" {
		return "vault.zip"
	}

	host := parsed.Host
	// Remove port if present.
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Replace non-alphanumeric characters with dashes.
	clean := nonAlphanumRE.ReplaceAllString(host, "-")
	clean = strings.Trim(clean, "-")

	if clean == "" {
		return "vault.zip"
	}

	return clean + "-vault.zip"
}
