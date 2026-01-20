package handlegithubcallback

import (
	"context"
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/githubauth"
	"trip2g/internal/logger"
	"trip2g/internal/oauthstate"
)

type Env interface {
	GetActiveGitHubOAuthCredentials(ctx context.Context) (db.GithubOauthCredential, error)
	DecryptData(ciphertext []byte) ([]byte, error)
	PublicURL() string
	Insecure() bool
	Logger() logger.Logger
	UserByEmail(ctx context.Context, email string) (db.User, error)
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
	creds, err := env.GetActiveGitHubOAuthCredentials(ctx)
	if err != nil || creds.ClientID == "" {
		env.Logger().Info("oauth login failed: oauth not configured",
			"provider", "github",
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_not_configured", http.StatusFound)
		return nil, nil //nolint:nilerr // error handled via redirect, not returned
	}

	// Decrypt client secret
	clientSecretBytes, err := env.DecryptData(creds.ClientSecretEncrypted)
	if err != nil {
		env.Logger().Error("oauth login failed: failed to decrypt client secret",
			"provider", "github",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}
	clientSecret := string(clientSecretBytes)

	// Check for OAuth error
	if errorParam := string(ctx.QueryArgs().Peek("error")); errorParam != "" {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "github",
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
			"provider", "github",
			"ip", clientIP)
		ctx.Redirect("/?berror=invalid_state", http.StatusFound)
		return nil, nil //nolint:nilerr // redirect response with error logged
	}

	// Exchange code for token
	code := string(ctx.QueryArgs().Peek("code"))

	tokenResp, err := githubauth.ExchangeCode(creds.ClientID, clientSecret, code)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "github",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Get primary verified email
	email, err := githubauth.GetPrimaryVerifiedEmail(tokenResp.AccessToken)
	if err != nil {
		env.Logger().Info("oauth login failed: email not verified",
			"provider", "github",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=email_not_verified", http.StatusFound)
		return nil, nil
	}

	// Find user by email
	user, err := env.UserByEmail(ctx, email)
	if err != nil {
		if db.IsNoFound(err) {
			env.Logger().Info("oauth login failed: user not found",
				"provider", "github",
				"email", email,
				"ip", clientIP)
			ctx.Redirect("/?berror=user_not_found", http.StatusFound)
			return nil, nil
		}
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "github",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	// Setup user token (JWT cookie)
	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		env.Logger().Info("oauth login failed: oauth error",
			"provider", "github",
			"error", err.Error(),
			"ip", clientIP)
		ctx.Redirect("/?berror=oauth_failed", http.StatusFound)
		return nil, nil
	}

	env.Logger().Info("oauth login success",
		"provider", "github",
		"email", email,
		"user_id", user.ID,
		"ip", clientIP)

	// Redirect to saved URL
	ctx.Redirect(redirect, http.StatusFound)
	return nil, nil
}

func (*Endpoint) Path() string {
	return "/_system/auth/github/callback"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
