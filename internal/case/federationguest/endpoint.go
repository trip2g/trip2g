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
	// not the session cookie, and no other part of the site reads it.
	NonceCookie = "trip2g_guest_nonce"
	// hostPrefix makes a browser refuse the cookie unless it is Secure, host-only
	// and at the root, so a page on a sibling subdomain cannot plant one.
	hostPrefix = "__Host-"

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
	nonce := string(req.Req.Request.Header.Cookie(nonceCookieName(env)))
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
		setWasGuestCookie(req.Req, env.Insecure(), true)
	}

	appresp.Redirect(req.Req, path, http.StatusFound)
	return nil, nil
}

// nonceCookieName takes the __Host- prefix whenever the cookie is Secure. The
// prefix needs Path=/, and a cookie path is not a boundary anyway: the ticket is
// what the nonce opens, and it is spent by one request.
func nonceCookieName(env Env) string {
	if env.Insecure() {
		return NonceCookie
	}
	return hostPrefix + NonceCookie
}

// setNonceCookie sets the nonce for the GET leg, or clears it when nonce is
// empty. Lax is what lets it ride that GET: a top-level navigation started by
// the peer's own 303, same-origin with the POST that set the cookie.
func setNonceCookie(ctx *fasthttp.RequestCtx, env Env, nonce string) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)

	c.SetKey(nonceCookieName(env))
	c.SetValue(nonce)
	c.SetPath("/")
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

	back := ""
	var refusal *Refusal
	if errors.As(err, &refusal) {
		back = refusal.Back
	}
	return &appreq.SystemMessageError{Code: code, Msg: msgRefused, Err: err, Back: back}
}

// WasGuestCookie remembers that this browser had a guest session, so that the
// sign-in wall can say why it is here once the session is gone. It carries no
// identity and no right.
const WasGuestCookie = "trip2g_was_guest"

// wasGuestTTL is how long the memory outlives the session it is about.
const wasGuestTTL = 30 * 24 * time.Hour

func setWasGuestCookie(ctx *fasthttp.RequestCtx, insecure, set bool) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)

	c.SetKey(WasGuestCookie)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(!insecure)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	if set {
		c.SetValue("1")
		c.SetMaxAge(int(wasGuestTTL / time.Second))
	} else {
		c.SetExpire(fasthttp.CookieExpireDelete)
	}
	ctx.Response.Header.SetCookie(c)
}

// TakeWasGuest reports whether the browser remembers a guest session, and
// forgets it: the wall tells the reader once.
func TakeWasGuest(ctx *fasthttp.RequestCtx, insecure bool) bool {
	if len(ctx.Request.Header.Cookie(WasGuestCookie)) == 0 {
		return false
	}

	setWasGuestCookie(ctx, insecure, false)
	return true
}

// ForgetWasGuest drops the memory of a guest session: a guest who left does
// not need to be told why they are gone.
func ForgetWasGuest(ctx *fasthttp.RequestCtx, insecure bool) {
	setWasGuestCookie(ctx, insecure, false)
}
