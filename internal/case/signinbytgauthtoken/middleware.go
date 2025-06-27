package signinbytgauthtoken

import (
	"net/http"
	"net/url"

	"github.com/valyala/fasthttp"
)

const QueryParam = "tg_auth_token"

func Process(ctx *fasthttp.RequestCtx, env Env) bool {
	token := string(ctx.QueryArgs().Peek(QueryParam))
	if token == "" {
		return false
	}

	err := Resolve(ctx, env, token)
	if err != nil {
		env.Logger().Error("failed to resolve sign-in by Telegram auth token", "error", err)
		ctx.SetStatusCode(http.StatusInternalServerError)
		return true
	}

	parsedURL, err := url.Parse(string(ctx.Request.Header.RequestURI()))
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		return true
	}

	// Validate redirect URL against trusted domains
	if !isValidRedirectURL(parsedURL, env.TrustedDomains()) {
		env.Logger().Warn("attempted redirect to untrusted domain", "host", parsedURL.Host)
		ctx.SetStatusCode(http.StatusBadRequest)
		return true
	}

	query := parsedURL.Query()
	query.Del(QueryParam)
	parsedURL.RawQuery = query.Encode()

	ctx.Redirect(parsedURL.String(), http.StatusFound)

	return true
}

func isValidRedirectURL(redirectURL *url.URL, trustedDomains []string) bool {
	// Allow relative paths (no host specified)
	if redirectURL.Host == "" {
		return true
	}

	// Check if host matches any trusted domain
	for _, domain := range trustedDomains {
		if redirectURL.Host == domain {
			return true
		}
	}

	return false
}
