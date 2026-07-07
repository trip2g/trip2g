package handleoidccallback

import (
	"context"
	"net/http"

	"trip2g/internal/appreq"
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
		ctx.Redirect("/?berror=oauth_not_configured", http.StatusFound)
		return nil, nil //nolint:nilerr // error handled via redirect, not returned
	}

	// Decrypt client secret
	clientSecretBytes, err := env.DecryptData(creds.ClientSecretEncrypted)
	if err != nil {
		env.Logger().Error("oauth login failed: failed to decrypt client secret",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}
	clientSecret := string(clientSecretBytes)

	// Check for OAuth error
	if errorParam := string(ctx.QueryArgs().Peek("error")); errorParam != "" {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", errorParam,
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Validate state (CSRF protection)
	stateParam := string(ctx.QueryArgs().Peek("state"))
	redirect, err := oauthstate.Validate(ctx, stateParam, env.Insecure())
	if err != nil {
		env.Logger().Info("oauth login failed: invalid state",
			"provider", "oidc",
			"ip", clientIP)
		ctx.Redirect("/?berror=invalid_state", http.StatusFound)
		return nil, nil //nolint:nilerr // redirect response with error logged
	}

	// Discover OIDC endpoints from issuer
	endpoints, err := oidcauth.Discover(creds.Issuer)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
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
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Verify the id_token signature against the provider's JWKS and its
	// standard claims (iss/aud/exp/iat) before trusting the token response.
	// Identity details below still come from userinfo (groups, verified email).
	if !verifyIDToken(env, ctx, creds, endpoints, tokenResp.IDToken, clientIP) {
		return nil, nil
	}

	// Get user info
	userInfo, err := oidcauth.GetUserInfo(tokenResp.AccessToken, endpoints.UserInfoEndpoint)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Find user by email
	user, err := env.UserByEmail(ctx, userInfo.Email)
	if err != nil && !db.IsNoFound(err) {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}
	exists := err == nil

	// Apply the configured access gate to EVERY login (existing and new users)
	if berr := accessBError(creds, userInfo); berr != "" {
		env.Logger().Info("oauth login failed: access denied",
			"provider", "oidc",
			"email", userInfo.Email,
			"ip", clientIP)
		ctx.Redirect("/?berror="+berr, http.StatusFound)
		return nil, nil
	}

	if !exists {
		// No existing user: decide whether to auto-provision
		if berr := provisionBError(creds, userInfo); berr != "" {
			reason := "access denied"
			if berr == berrUserNotFound {
				reason = berrUserNotFound
			}
			env.Logger().Info("oauth login failed: "+reason,
				"provider", "oidc",
				"email", userInfo.Email,
				"ip", clientIP)
			ctx.Redirect("/?berror="+berr, http.StatusFound)
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
			ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
			return nil, nil
		}

		env.Logger().Info("oauth user provisioned",
			"provider", "oidc",
			"email", userInfo.Email,
			"user_id", user.ID)
	}

	// Setup user token (JWT cookie)
	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	env.Logger().Info("oauth login success",
		"provider", "oidc",
		"email", userInfo.Email,
		"user_id", user.ID,
		"ip", clientIP)

	// Redirect to saved URL
	ctx.Redirect(redirect, http.StatusFound)
	return nil, nil
}

// verifyIDToken checks the id_token signature + standard claims. On failure it
// logs, redirects with an error, and returns false so the caller aborts.
func verifyIDToken(env Env, ctx *fasthttp.RequestCtx, creds db.OidcCredential, endpoints oidcauth.Endpoints, idToken, clientIP string) bool {
	_, err := env.VerifyOIDCIDToken(ctx, idToken, oidcauth.VerifyParams{
		JWKSURI:  endpoints.JWKSURI,
		Issuer:   creds.Issuer,
		ClientID: creds.ClientID,
	})
	if err != nil {
		env.Logger().Info("oauth login failed: id_token verification failed",
			"provider", "oidc",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return false
	}
	return true
}

func (*Endpoint) Path() string {
	return "/_system/auth/oidc/callback"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
