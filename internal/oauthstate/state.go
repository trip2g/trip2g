package oauthstate

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"time"

	"trip2g/internal/appresp"

	"github.com/valyala/fasthttp"
)

// CookieName is shared across all OAuth providers.
// TODO: refactor to accept cookieName as parameter to support concurrent OAuth flows
// from multiple providers (e.g. Google + LinkedIn) without cookie collision.
const CookieName = "oauth_state"

var (
	ErrInvalidState = errors.New("invalid oauth state")
	ErrStateMissing = errors.New("oauth state missing")
)

type State struct {
	Redirect string `json:"r"`
	Nonce    string `json:"n"`
	// OIDCNonce is the OpenID Connect nonce echoed in the id_token; empty for
	// providers (Google, GitHub) that don't use it.
	OIDCNonce string `json:"on,omitempty"`
}

// safeRedirect ensures the redirect target is a same-origin relative path.
// An absolute URL on host — what the frontend sends, since it reads
// location.href — is reduced to its path first, and appresp.SamePath then
// decides. A path can itself be another origin, so that rule is the one every
// shape passes and this package does not keep a second copy of it.
func safeRedirect(host, redirect string) string {
	if scheme.MatchString(redirect) {
		redirect = ownOriginPath(host, redirect)
	}

	return appresp.SamePath(redirect)
}

var scheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// ownOriginPath reduces an absolute URL that names this very host to its path,
// and answers "" for every other one, which SamePath turns into "/". The
// scheme is not compared: behind a TLS-terminating proxy the page and the
// request disagree about it, and the answer is a path either way.
func ownOriginPath(host, redirect string) string {
	if host == "" {
		return ""
	}

	target, err := url.Parse(redirect)
	if err != nil || target.Host != host {
		return ""
	}

	path := target.EscapedPath()
	if target.RawQuery != "" {
		path += "?" + target.RawQuery
	}

	if target.Fragment != "" {
		path += "#" + target.EscapedFragment()
	}

	return path
}

// Generate creates new state, sets cookie, returns encoded state for OAuth URL.
func Generate(ctx *fasthttp.RequestCtx, redirect string, insecure bool) (string, error) {
	return generate(ctx, redirect, "", insecure)
}

// GenerateWithOIDCNonce is like Generate but also embeds an OpenID Connect nonce
// in the signed state blob so the callback can bind it to the id_token's nonce
// claim (OIDC replay protection).
func GenerateWithOIDCNonce(ctx *fasthttp.RequestCtx, redirect, oidcNonce string, insecure bool) (string, error) {
	return generate(ctx, redirect, oidcNonce, insecure)
}

func generate(ctx *fasthttp.RequestCtx, redirect, oidcNonce string, insecure bool) (string, error) {
	redirect = safeRedirect(string(ctx.Host()), redirect)
	// Generate random nonce (16 bytes, hex encoded = 32 chars)
	nonceBytes := make([]byte, 16)
	_, err := rand.Read(nonceBytes)
	if err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(nonceBytes)

	// Create state
	state := State{
		Redirect:  redirect,
		Nonce:     nonce,
		OIDCNonce: oidcNonce,
	}

	// Encode state as JSON then base64
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	encodedState := base64.URLEncoding.EncodeToString(stateJSON)

	// Set cookie with nonce only (for CSRF validation)
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)

	c.SetKey(CookieName)
	c.SetValue(nonce)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(!insecure)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	c.SetMaxAge(300) // 5 minutes

	ctx.Response.Header.SetCookie(c)

	return encodedState, nil
}

// Validate checks state param against cookie, returns redirect URL.
// Deletes cookie after validation.
func Validate(ctx *fasthttp.RequestCtx, stateParam string, insecure bool) (string, error) {
	redirect, _, err := validate(ctx, stateParam, insecure)
	return redirect, err
}

// ValidateWithOIDCNonce validates like Validate and additionally returns the
// OpenID Connect nonce embedded in the state blob (empty when none was set).
// Returns (redirect, oidcNonce, error).
func ValidateWithOIDCNonce(ctx *fasthttp.RequestCtx, stateParam string, insecure bool) (string, string, error) {
	return validate(ctx, stateParam, insecure)
}

func validate(ctx *fasthttp.RequestCtx, stateParam string, insecure bool) (string, string, error) {
	// Get nonce from cookie
	cookieNonce := string(ctx.Request.Header.Cookie(CookieName))
	if cookieNonce == "" {
		return "", "", ErrStateMissing
	}

	// Delete cookie immediately
	deleteCookie(ctx, insecure)

	// Decode state param
	stateJSON, err := base64.URLEncoding.DecodeString(stateParam)
	if err != nil {
		return "", "", ErrInvalidState
	}

	var state State
	if err = json.Unmarshal(stateJSON, &state); err != nil {
		return "", "", ErrInvalidState
	}

	// Validate nonce matches
	if state.Nonce != cookieNonce {
		return "", "", ErrInvalidState
	}

	// Return redirect URL (default to "/" if empty)
	if state.Redirect == "" {
		return "/", state.OIDCNonce, nil
	}

	return state.Redirect, state.OIDCNonce, nil
}

func deleteCookie(ctx *fasthttp.RequestCtx, insecure bool) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)

	c.SetKey(CookieName)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	c.SetSecure(!insecure)
	c.SetExpire(time.Now().Add(-time.Hour))

	ctx.Response.Header.SetCookie(c)
}
