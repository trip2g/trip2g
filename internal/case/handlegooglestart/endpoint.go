package handlegooglestart

import (
	"context"
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/googleauth"
	"trip2g/internal/oauthstate"
)

type Env interface {
	GetActiveGoogleOAuthCredentials(ctx context.Context) (db.GoogleOauthCredential, error)
	PublicURL() string
	Insecure() bool
}

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req

	// Load credentials from DB
	creds, err := env.GetActiveGoogleOAuthCredentials(ctx)
	if err != nil || creds.ClientID == "" {
		req.Req.Redirect("/?berror=oauth_not_configured", http.StatusFound)
		return nil, nil
	}

	// Get redirect URL from query params (default to "/")
	redirect := string(req.Req.QueryArgs().Peek("redirect"))
	if redirect == "" {
		redirect = "/"
	}

	// Generate state with CSRF nonce
	state, err := oauthstate.Generate(req.Req, redirect, env.Insecure())
	if err != nil {
		req.Req.SetStatusCode(http.StatusInternalServerError)
		return nil, nil //nolint:nilerr // redirect response, error logged elsewhere
	}

	// Build callback URL
	callbackURL := env.PublicURL() + "/_system/auth/google/callback"

	// Redirect to Google OAuth
	authURL := googleauth.BuildAuthURL(creds.ClientID, callbackURL, state)
	req.Req.Redirect(authURL, http.StatusFound)

	return nil, nil
}

func (*Endpoint) Path() string {
	return "/_system/auth/google"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
