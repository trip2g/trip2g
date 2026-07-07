package main

import (
	"context"
	"fmt"
	"net/url"
	"time"
	"trip2g/internal/configregistry"
	"trip2g/internal/db"
	"trip2g/internal/githubauth"
	"trip2g/internal/googleauth"
	"trip2g/internal/oidcauth"
)

// envOIDCCredential builds a virtual OIDC credential from the OIDC_* env vars.
// It is never persisted: set the env vars and it appears as the active provider;
// remove them and it is gone on the next boot. Editing or deleting it via the
// admin API hits no DB row (id 0) and harmlessly no-ops.
func (a *app) envOIDCCredential() (db.OidcCredential, bool) {
	o := a.config.OIDC
	if o.Issuer == "" || o.ClientID == "" || o.ClientSecret == "" {
		return db.OidcCredential{}, false
	}
	enc, err := a.EncryptData([]byte(o.ClientSecret))
	if err != nil {
		a.log.Error("failed to encrypt env OIDC client secret", "error", err.Error())
		return db.OidcCredential{}, false
	}
	scopes := o.Scopes
	if scopes == "" {
		scopes = "openid email profile"
	}
	return db.OidcCredential{
		ID:                    0, // sentinel: virtual, env-managed (not in DB)
		Name:                  "env",
		Issuer:                o.Issuer,
		ClientID:              o.ClientID,
		ClientSecretEncrypted: enc,
		Scopes:                scopes,
		AutoProvision:         o.AutoProvision,
		AllowedEmailDomain:    o.AllowedEmailDomain,
		RequiredGroup:         o.RequiredGroup,
		Active:                true,
		CreatedAt:             time.Now(),
	}, true
}

// GetActiveOIDCCredentials shadows the generated query: the env-managed provider
// (if configured) always wins; otherwise fall back to the DB-stored active row.
func (a *app) GetActiveOIDCCredentials(ctx context.Context) (db.OidcCredential, error) {
	if cred, ok := a.envOIDCCredential(); ok {
		return cred, nil
	}
	return a.Queries.GetActiveOIDCCredentials(ctx)
}

// ListOIDCCredentials shadows the generated query so the admin UI also lists the
// env-managed provider (read-only) alongside any DB-stored ones.
func (a *app) ListOIDCCredentials(ctx context.Context) ([]db.OidcCredential, error) {
	rows, err := a.Queries.ListOIDCCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if cred, ok := a.envOIDCCredential(); ok {
		// give the virtual row a real created_by so the GraphQL createdBy field resolves
		if owner, e := a.UserByEmail(ctx, a.config.OwnerEmail); e == nil {
			cred.CreatedBy = owner.ID
		}
		rows = append([]db.OidcCredential{cred}, rows...)
	}
	return rows, nil
}

// BuildGoogleAuthURL returns (callbackURL, authURL, error).
// callbackURL is always returned for admin UI display.
// authURL is only returned if OAuth is configured (or dry=true for just getting callbackURL).
//
//nolint:nonamedreturns // named returns document the multiple string return values
func (a *app) BuildGoogleAuthURL(ctx context.Context, redirectURL string, dry bool) (callbackURL string, authURL string, err error) {
	publicURL := a.GetPublicURLForRequest(ctx)
	callbackURL = fmt.Sprintf("%s/_system/auth/google/callback", publicURL)

	if dry {
		return callbackURL, "", nil
	}

	creds, err := a.GetActiveGoogleOAuthCredentials(ctx)
	if err != nil {
		// No active credentials - OAuth not configured
		return callbackURL, "", nil //nolint:nilerr // expected: missing credentials returns empty authURL
	}
	if creds.ClientID == "" {
		return callbackURL, "", nil
	}

	authURL = fmt.Sprintf("%s/_system/auth/google?redirect=%s", publicURL, url.QueryEscape(redirectURL))
	return callbackURL, authURL, nil
}

// BuildOIDCAuthURL returns (callbackURL, authURL, error).
// callbackURL is always returned for admin UI display.
// authURL is only returned if OIDC is configured (or dry=true for just getting callbackURL).
//
//nolint:nonamedreturns // named returns document the multiple string return values
func (a *app) BuildOIDCAuthURL(ctx context.Context, redirectURL string, dry bool) (callbackURL string, authURL string, err error) {
	publicURL := a.GetPublicURLForRequest(ctx)
	callbackURL = fmt.Sprintf("%s/_system/auth/oidc/callback", publicURL)

	if dry {
		return callbackURL, "", nil
	}

	creds, err := a.GetActiveOIDCCredentials(ctx)
	if err != nil {
		// No active credentials - OIDC not configured
		return callbackURL, "", nil //nolint:nilerr // expected: missing credentials returns empty authURL
	}
	if creds.ClientID == "" {
		return callbackURL, "", nil
	}

	authURL = fmt.Sprintf("%s/_system/auth/oidc?redirect=%s", publicURL, url.QueryEscape(redirectURL))
	return callbackURL, authURL, nil
}

// BuildGitHubAuthURL returns (callbackURL, authURL, error).
// callbackURL is always returned for admin UI display.
// authURL is only returned if OAuth is configured (or dry=true for just getting callbackURL).
//
//nolint:nonamedreturns // named returns document the multiple string return values
func (a *app) BuildGitHubAuthURL(ctx context.Context, redirectURL string, dry bool) (callbackURL string, authURL string, err error) {
	publicURL := a.GetPublicURLForRequest(ctx)
	callbackURL = fmt.Sprintf("%s/_system/auth/github/callback", publicURL)

	if dry {
		return callbackURL, "", nil
	}

	creds, err := a.GetActiveGitHubOAuthCredentials(ctx)
	if err != nil {
		// No active credentials - OAuth not configured
		return callbackURL, "", nil //nolint:nilerr // expected: missing credentials returns empty authURL
	}
	if creds.ClientID == "" {
		return callbackURL, "", nil
	}

	authURL = fmt.Sprintf("%s/_system/auth/github?redirect=%s", publicURL, url.QueryEscape(redirectURL))
	return callbackURL, authURL, nil
}

// ValidateGoogleOAuthCredentials validates Google OAuth credentials by making a test API call.
func (a *app) ValidateGoogleOAuthCredentials(ctx context.Context, clientID, clientSecret string) error {
	redirectURI := fmt.Sprintf("%s/_system/auth/google/callback", a.GetPublicURLForRequest(ctx))
	return googleauth.ValidateCredentials(clientID, clientSecret, redirectURI)
}

// ValidateOIDCCredentials probes that the issuer's discovery document is
// reachable. It intentionally does NOT verify clientID/clientSecret — there is
// no cheap check without a full auth flow; bad client creds surface at first login.
func (a *app) ValidateOIDCCredentials(ctx context.Context, issuer, clientID, clientSecret string) error {
	_ = clientID
	_ = clientSecret
	_, err := oidcauth.Discover(issuer)
	return err
}

// VerifyOIDCIDToken verifies an OIDC id_token signature against the provider's
// JWKS and standard claims, caching keys across logins. Rejection is a
// validation error (auth denied), never a panic.
func (a *app) VerifyOIDCIDToken(ctx context.Context, rawIDToken string, p oidcauth.VerifyParams) (*oidcauth.IDTokenClaims, error) {
	return a.oidcKeys.VerifyIDToken(ctx, rawIDToken, p)
}

// ValidateGitHubOAuthCredentials validates GitHub OAuth credentials by making a test API call.
func (a *app) ValidateGitHubOAuthCredentials(ctx context.Context, clientID, clientSecret string) error {
	return githubauth.ValidateCredentials(clientID, clientSecret)
}

func (a *app) TurnstileSiteKey() string {
	return a.config.Turnstile.SiteKey
}

func (a *app) VerifyTurnstile(ctx context.Context, token, ip string) error {
	return a.Client.VerifyCaptcha(ctx, token, ip)
}

func (a *app) IncrementAndCheckSigninCounter() bool {
	threshold := a.CaptchaSigninThreshold()
	return a.signinCounter.IncrementAndCheck(threshold)
}

func (a *app) CaptchaSigninThreshold() int64 {
	row, err := a.queries.GetLatestConfigInt(context.Background(), configregistry.ConfigCaptchaSigninThreshold)
	if err != nil {
		return 5 // default
	}
	return row.Value
}
