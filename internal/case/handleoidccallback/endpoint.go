package handleoidccallback

import (
	"context"
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/oauthstate"
	"trip2g/internal/oidcauth"

	"github.com/valyala/fasthttp"
)

type Env interface {
	GetActiveOIDCCredentials(ctx context.Context) (db.OidcCredential, error)
	VerifyOIDCIDToken(ctx context.Context, rawIDToken string, p oidcauth.VerifyParams) (*oidcauth.IDTokenClaims, error)
	DecryptData(ciphertext []byte) ([]byte, error)
	PublicURL() string
	Insecure() bool
	Logger() logger.Logger
	UserByEmail(ctx context.Context, email string) (db.User, error)
	InsertUserWithEmail(ctx context.Context, arg db.InsertUserWithEmailParams) (db.User, error)
	SetupUserToken(ctx context.Context, userID int64) (string, error)
}

type Endpoint struct{}

//nolint:funlen // linear OAuth callback flow: discover → exchange → verify → userinfo → provision
func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req

	// Get client IP for logging
	clientIP := string(ctx.Request.Header.Peek("X-Forwarded-For"))
	if clientIP == "" {
		clientIP = ctx.RemoteIP().String()
	}

	// Load credentials from DB
	creds, err := env.GetActiveOIDCCredentials(ctx)
	if err != nil || creds.ClientID == "" {
		env.Logger().Info("oauth login failed: oauth not configured",
			"provider", "oidc",
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_not_configured", http.StatusFound)
		return nil, nil //nolint:nilerr // error handled via redirect, not returned
	}

	// Decrypt client secret
	clientSecretBytes, err := env.DecryptData(creds.ClientSecretEncrypted)
	if err != nil {
		env.Logger().Error("oauth login failed: failed to decrypt client secret",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}
	clientSecret := string(clientSecretBytes)

	// Check for OAuth error
	if errorParam := string(ctx.QueryArgs().Peek("error")); errorParam != "" {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", errorParam,
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Validate state (CSRF protection) and recover the OIDC nonce for replay
	// protection binding below.
	stateParam := string(ctx.QueryArgs().Peek("state"))
	redirect, nonce, err := oauthstate.ValidateWithOIDCNonce(ctx, stateParam, env.Insecure())
	if err != nil {
		env.Logger().Info("oauth login failed: invalid state",
			"provider", "oidc",
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=invalid_state", http.StatusFound)
		return nil, nil //nolint:nilerr // redirect response with error logged
	}

	// Discover OIDC endpoints from issuer
	endpoints, err := oidcauth.Discover(creds.Issuer)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Exchange code for token
	code := string(ctx.QueryArgs().Peek("code"))
	callbackURL := env.PublicURL() + "/_system/auth/oidc/callback"

	tokenResp, err := oidcauth.ExchangeCode(creds.ClientID, clientSecret, code, callbackURL, endpoints.TokenEndpoint)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Verify the id_token signature against the provider's JWKS and its
	// standard claims (iss/aud/exp, iat when present, plus nonce) before trusting
	// the token response. Identity details below still come from userinfo, but
	// its subject is bound to the verified id_token subject.
	idClaims, ok := verifyIDToken(env, ctx, creds, endpoints, tokenResp.IDToken, nonce, clientIP)
	if !ok {
		return nil, nil
	}

	// Get user info
	userInfo, err := oidcauth.GetUserInfo(tokenResp.AccessToken, endpoints.UserInfoEndpoint)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Bind userinfo to the verified id_token via sub (OIDC Core 5.3.2 MUST).
	if !subjectBound(idClaims.Subject, userInfo.Sub) {
		env.Logger().Info("oauth login failed: id_token/userinfo sub mismatch",
			"provider", "oidc",
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	verification := resolveEmailVerification(idClaims, userInfo)

	// Reject unverified identities before looking up a local account by email.
	// An email claim that has not been verified by the provider must never bind
	// an OIDC login to an existing user.
	user, berr, err := lookupAuthorizedUser(ctx, env, creds, userInfo, verification)
	if berr != "" {
		reason := berr
		if berr == berrEmailNotVerified {
			reason = verification.Reason
		}
		env.Logger().Info("oauth login failed: access denied",
			"provider", "oidc",
			"email", userInfo.Email,
			"berror", berr,
			"reason", reason,
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror="+berr, http.StatusFound)
		return nil, nil
	}

	if err != nil && !db.IsNoFound(err) {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}
	exists := err == nil

	if !exists {
		// No existing user: decide whether to auto-provision
		if pberr := provisionBError(creds, verification); pberr != "" {
			reason := "access denied"
			if pberr == berrUserNotFound {
				reason = berrUserNotFound
			}
			env.Logger().Info("oauth login failed: "+reason,
				"provider", "oidc",
				"email", userInfo.Email,
				"ip", clientIP)
			appresp.Redirect(ctx, "/?berror="+pberr, http.StatusFound)
			return nil, nil
		}

		// Provision a new user
		user, err = env.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{
			Email:      userInfo.Email,
			CreatedVia: "oidc",
		})
		if err != nil {
			env.Logger().Info("oauth login failed: oauth error",
				"provider", "oidc",
				"error", err.Error(),
				"ip", clientIP)
			appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
			return nil, nil
		}

		env.Logger().Info("oauth user provisioned",
			"provider", "oidc",
			"email", userInfo.Email,
			"email_verification_source", verification.Source,
			"user_id", user.ID)
	}

	// Setup user token (JWT cookie)
	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	env.Logger().Info("oauth login success",
		"provider", "oidc",
		"email", userInfo.Email,
		"email_verification_source", verification.Source,
		"user_id", user.ID,
		"ip", clientIP)

	// Redirect to saved URL
	appresp.Redirect(ctx, redirect, http.StatusFound)
	return nil, nil
}

// verifyIDToken checks the id_token signature + standard claims (including the
// nonce). On success it returns the verified claims; on failure it logs,
// redirects with an error, and returns ok=false so the caller aborts.
func verifyIDToken(
	env Env,
	ctx *fasthttp.RequestCtx,
	creds db.OidcCredential,
	endpoints oidcauth.Endpoints,
	idToken, nonce, clientIP string,
) (*oidcauth.IDTokenClaims, bool) {
	claims, err := env.VerifyOIDCIDToken(ctx, idToken, oidcauth.VerifyParams{
		JWKSURI:  endpoints.JWKSURI,
		Issuer:   creds.Issuer,
		ClientID: creds.ClientID,
		Nonce:    nonce,
	})
	if err != nil {
		env.Logger().Info("oauth login failed: id_token verification failed",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		appresp.Redirect(ctx, "/?berror=oauth_failed", http.StatusFound)
		return nil, false
	}
	return claims, true
}

func (*Endpoint) Path() string {
	return "/_system/auth/oidc/callback"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
