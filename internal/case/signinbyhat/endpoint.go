package signinbyhat

import (
	"context"
	"fmt"
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

type Env interface {
	ParseHotAuthToken(ctx context.Context, token string) (*model.HotAuthToken, error)
	UserByEmail(ctx context.Context, email string) (db.User, error)
	InsertUserWithEmail(ctx context.Context, params db.InsertUserWithEmailParams) (db.User, error)
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
	InsertAdmin(ctx context.Context, params db.InsertAdminParams) (db.Admin, error)
	SetupUserToken(ctx context.Context, userID int64) (string, error)
}

type Endpoint struct{}

func (e *Endpoint) Path() string {
	return "/_system/hat"
}

func (e *Endpoint) Method() string {
	return http.MethodPost
}

func (e *Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token := string(req.Req.PostArgs().Peek("token"))
	if token == "" {
		req.Req.SetStatusCode(http.StatusBadRequest)
		req.Req.SetBodyString("missing token")
		return nil, nil
	}

	redirect, err := Resolve(req.Req, req.Env.(Env), token)
	if err != nil {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		req.Req.SetBodyString(fmt.Sprintf("authentication failed: %v", err))
		return nil, nil
	}

	req.Req.SetStatusCode(http.StatusFound)
	req.Req.Response.Header.Set("Location", location(redirect, "/"))
	return nil, nil
}

// GetEndpoint makes the HAT exchange clickable from a plain browser link (e.g.
// the trip2g-server login-link CLI output). Same Resolve, GET on the query
// string instead of POST form data, redirects to /admin instead of /.
type GetEndpoint struct{}

func (e *GetEndpoint) Path() string {
	return "/_system/hat"
}

func (e *GetEndpoint) Method() string {
	return http.MethodGet
}

func (e *GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	// The token ends up in access logs and browser history via the query
	// string; accepted tradeoff for a 5-minute one-time bootstrap link.
	token := string(req.Req.QueryArgs().Peek("token"))
	if token == "" {
		req.Req.SetStatusCode(http.StatusBadRequest)
		req.Req.SetBodyString("missing token")
		return nil, nil
	}

	redirect, err := Resolve(req.Req, req.Env.(Env), token)
	if err != nil {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		req.Req.SetBodyString(fmt.Sprintf("authentication failed: %v", err))
		return nil, nil
	}

	req.Req.SetStatusCode(http.StatusFound)
	req.Req.Response.Header.Set("Location", location(redirect, "/admin"))
	return nil, nil
}

// The redirect rides the signed token, so it cannot be steered by whoever opens
// the link; a link that names none keeps the endpoint's own default.
func location(redirect, fallback string) string {
	if redirect == "" {
		return fallback
	}

	return redirect
}

// Resolve signs the token's subject in and reports where to send the browser;
// an empty redirect leaves the caller's own default.
//
// Only a provisioning token creates the user or touches roles. An ordinary
// sign-in link is a way in for someone who already exists, so an unknown
// address is refused rather than silently becoming an account.
func Resolve(ctx context.Context, env Env, token string) (string, error) {
	hotAuthToken, err := env.ParseHotAuthToken(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	user, err := env.UserByEmail(ctx, hotAuthToken.Email)
	if err != nil {
		if !db.IsNoFound(err) {
			return "", fmt.Errorf("failed to get user: %w", err)
		}

		if !hotAuthToken.AdminEnter {
			return "", fmt.Errorf("no user with email %s", hotAuthToken.Email)
		}

		params := db.InsertUserWithEmailParams{
			Email:      hotAuthToken.Email,
			CreatedVia: "hot_auth_token",
		}

		user, err = env.InsertUserWithEmail(ctx, params)
		if err != nil {
			return "", fmt.Errorf("failed to create user: %w", err)
		}
	}

	if hotAuthToken.AdminEnter {
		err = ensureUserIsAdmin(ctx, env, user.ID)
		if err != nil {
			return "", err
		}
	}

	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return hotAuthToken.Redirect, nil
}

func ensureUserIsAdmin(ctx context.Context, env Env, userID int64) error {
	_, err := env.AdminByUserID(ctx, userID)
	if err == nil {
		return nil
	}

	if !db.IsNoFound(err) {
		return fmt.Errorf("failed to check admin status: %w", err)
	}

	_, insertErr := env.InsertAdmin(ctx, db.InsertAdminParams{UserID: userID})
	if insertErr != nil {
		return fmt.Errorf("failed to make user admin: %w", insertErr)
	}

	return nil
}
