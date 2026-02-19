package oauthstate

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

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
}

// Generate creates new state, sets cookie, returns encoded state for OAuth URL.
func Generate(ctx *fasthttp.RequestCtx, redirect string, insecure bool) (string, error) {
	// Generate random nonce (16 bytes, hex encoded = 32 chars)
	nonceBytes := make([]byte, 16)
	_, err := rand.Read(nonceBytes)
	if err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(nonceBytes)

	// Create state
	state := State{
		Redirect: redirect,
		Nonce:    nonce,
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
	// Get nonce from cookie
	cookieNonce := string(ctx.Request.Header.Cookie(CookieName))
	if cookieNonce == "" {
		return "", ErrStateMissing
	}

	// Delete cookie immediately
	deleteCookie(ctx, insecure)

	// Decode state param
	stateJSON, err := base64.URLEncoding.DecodeString(stateParam)
	if err != nil {
		return "", ErrInvalidState
	}

	var state State
	err = json.Unmarshal(stateJSON, &state)
	if err != nil {
		return "", ErrInvalidState
	}

	// Validate nonce matches
	if state.Nonce != cookieNonce {
		return "", ErrInvalidState
	}

	// Return redirect URL (default to "/" if empty)
	if state.Redirect == "" {
		return "/", nil
	}

	return state.Redirect, nil
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
