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

	query := parsedURL.Query()
	query.Del(QueryParam)
	parsedURL.RawQuery = query.Encode()

	ctx.Redirect(parsedURL.String(), http.StatusFound)

	return true
}
