package handleoidcstart

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/oauthstate"
	"trip2g/internal/oidcauth"
)

type Env interface {
	GetActiveOIDCCredentials(ctx context.Context) (db.OidcCredential, error)
	PublicURL() string
	Insecure() bool
}

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req

	// Load credentials from DB
	creds, err := env.GetActiveOIDCCredentials(ctx)
	if err != nil || creds.ClientID == "" {
		req.Req.Redirect("/?berror=oauth_not_configured", http.StatusFound)
		return nil, nil //nolint:nilerr // error handled via redirect, not returned
	}

	// Get redirect URL from query params (default to "/")
	redirect := string(req.Req.QueryArgs().Peek("redirect"))
	if redirect == "" {
		redirect = "/"
	}

	// Generate the OIDC nonce (replay protection): sent in the auth request and
	// carried through the signed state blob so the callback can bind it to the
	// id_token's nonce claim.
	nonceBytes := make([]byte, 16)
	if _, err = rand.Read(nonceBytes); err != nil {
		req.Req.SetStatusCode(http.StatusInternalServerError)
		return nil, nil //nolint:nilerr // redirect response, error logged elsewhere
	}
	nonce := hex.EncodeToString(nonceBytes)

	// Generate state with CSRF nonce + embedded OIDC nonce
	state, err := oauthstate.GenerateWithOIDCNonce(req.Req, redirect, nonce, env.Insecure())
	if err != nil {
		req.Req.SetStatusCode(http.StatusInternalServerError)
		return nil, nil //nolint:nilerr // redirect response, error logged elsewhere
	}

	// Discover OIDC endpoints from issuer
	endpoints, err := oidcauth.Discover(creds.Issuer)
	if err != nil {
		req.Req.Redirect("/?berror=oauth_not_configured", http.StatusFound)
		return nil, nil //nolint:nilerr // error handled via redirect, not returned
	}

	// Build callback URL
	callbackURL := env.PublicURL() + "/_system/auth/oidc/callback"

	// Redirect to OIDC provider
	authURL := oidcauth.BuildAuthURL(endpoints.AuthorizationEndpoint, creds.ClientID, callbackURL, state, creds.Scopes, nonce)
	req.Req.Redirect(authURL, http.StatusFound)

	return nil, nil
}

func (*Endpoint) Path() string {
	return "/_system/auth/oidc"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
