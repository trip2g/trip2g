package federationguest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/appresp"

	"github.com/valyala/fasthttp"
)

const (
	Path        = "/_system/federation/guest"
	TicketParam = "ticket"
	// NonceCookie binds a ticket to the browser that posted the handoff. It is
	// not the session cookie, and only the guest endpoint ever sees it.
	NonceCookie = "trip2g_guest_nonce"

	msgRefused = "federation_guest_refused"
)

// Endpoint takes the handoff the hub's page posts and answers with the ticket
// for the GET leg.
type Endpoint struct{}

func (*Endpoint) Path() string   { return Path }
func (*Endpoint) Method() string { return http.MethodPost }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	form := Form{
		Token:  string(req.Req.PostArgs().Peek("token")),
		Path:   string(req.Req.PostArgs().Peek("path")),
		Origin: string(req.Req.Request.Header.Peek("Origin")),
	}

	env := req.Env.(Env)
	admitted, err := Admit(req.Req, env, form)
	if err != nil {
		return nil, failure(err)
	}

	setNonceCookie(req.Req, env, admitted.Nonce)
	query := url.Values{TicketParam: {admitted.Ticket}}
	appresp.Redirect(req.Req, Path+"?"+query.Encode(), http.StatusSeeOther)
	return nil, nil
}

// GetEndpoint exchanges a ticket for a guest session, unless the browser
// already holds a session of its own here: that one wins and is left alone.
type GetEndpoint struct{}

func (*GetEndpoint) Path() string   { return Path }
func (*GetEndpoint) Method() string { return http.MethodGet }

func (*GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	nonce := string(req.Req.Request.Header.Cookie(NonceCookie))
	// Cleared whatever happens next: a nonce serves one attempt.
	setNonceCookie(req.Req, env, "")

	data, path, err := Enter(env, string(req.Req.QueryArgs().Peek(TicketParam)), nonce)
	if err != nil {
		return nil, failure(err)
	}

	current, err := req.UserToken()
	if err != nil {
		return nil, fmt.Errorf("read current session: %w", err)
	}

	if current == nil || current.IsFederationGuest() {
		stored, storeErr := req.TokenManager.Store(req.Req, data)
		if storeErr != nil {
			return nil, fmt.Errorf("store guest session: %w", storeErr)
		}
		req.SetUserToken(&stored.Data)
	}

	appresp.Redirect(req.Req, path, http.StatusFound)
	return nil, nil
}

// setNonceCookie sets the nonce for the GET leg, or clears it when nonce is
// empty. Lax is what lets it ride that GET: a top-level navigation started by
// the peer's own 303, same-origin with the POST that set the cookie.
func setNonceCookie(ctx *fasthttp.RequestCtx, env Env, nonce string) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)

	c.SetKey(NonceCookie)
	c.SetValue(nonce)
	c.SetPath(Path)
	c.SetHTTPOnly(true)
	c.SetSecure(!env.Insecure())
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	if nonce == "" {
		c.SetExpire(fasthttp.CookieExpireDelete)
	} else {
		c.SetMaxAge(int(TicketTTL / time.Second))
	}
	ctx.Response.Header.SetCookie(c)
}

func failure(err error) error {
	code := http.StatusInternalServerError
	if errors.Is(err, ErrRefused) {
		code = http.StatusForbidden
	}
	return &appreq.SystemMessageError{Code: code, Msg: msgRefused, Err: err}
}
