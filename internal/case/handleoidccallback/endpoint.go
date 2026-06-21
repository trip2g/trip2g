package handleoidccallback

import (
	"context"
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/oauthstate"
	"trip2g/internal/oidcauth"
)

type Env interface {
	GetActiveOIDCCredentials(ctx context.Context) (db.OidcCredential, error)
	DecryptData(ciphertext []byte) ([]byte, error)
	PublicURL() string
	Insecure() bool
	Logger() logger.Logger
	UserByEmail(ctx context.Context, email string) (db.User, error)
	InsertUserWithEmail(ctx context.Context, arg db.InsertUserWithEmailParams) (db.User, error)
	SetupUserToken(ctx context.Context, userID int64) (string, error)
}

type Endpoint struct{}

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
	if err != nil {
		if db.IsNoFound(err) {
			// No existing user: apply account policy
			outcome := decideAccount(creds, userInfo)
			if outcome.Reject {
				env.Logger().Info("oauth login failed: "+outcome.BError,
					"provider", "oidc",
					"email", userInfo.Email,
					"ip", clientIP)
				ctx.Redirect("/?berror="+outcome.BError, http.StatusFound)
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
		} else {
			env.Logger().Info("oauth login failed: oauth error",
				"provider", "oidc",
				"error", err.Error(),
				"ip", clientIP)
			ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
			return nil, nil
		}
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

func (*Endpoint) Path() string {
	return "/_system/auth/oidc/callback"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
