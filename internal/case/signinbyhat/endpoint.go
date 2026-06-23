package signinbyhat

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/valyala/fasthttp"
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

	err := Resolve(req.Req, req.Env.(Env), token)
	if err != nil {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		req.Req.SetBodyString(fmt.Sprintf("authentication failed: %v", err))
		return nil, nil
	}

	req.Req.SetStatusCode(http.StatusFound)
	req.Req.Response.Header.Set("Location", "/")
	return nil, nil
}

// GetEndpoint serves the hot-auth-token sign-in over GET so the token can be a
// single clickable login link, in addition to the POST form.
//
// Tradeoff vs the POST Endpoint: a GET link carries the token in the URL, so it
// leaks into browser history and server access logs. This is acceptable here
// because the token is short-lived and this is localhost/personal use.
type GetEndpoint struct{}

func (e *GetEndpoint) Path() string {
	return "/_system/hat"
}

func (e *GetEndpoint) Method() string {
	return http.MethodGet
}

func (e *GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	token := string(req.Req.URI().QueryArgs().Peek("token"))
	if token == "" {
		req.Req.SetStatusCode(http.StatusBadRequest)
		req.Req.SetBodyString("missing token")
		return nil, nil
	}

	err := Resolve(req.Req, req.Env.(Env), token)
	if err != nil {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		req.Req.SetBodyString(fmt.Sprintf("authentication failed: %v", err))
		return nil, nil
	}

	req.Req.SetStatusCode(http.StatusFound)
	req.Req.Response.Header.Set("Location", sanitizeRedirect(req.Req.URI().QueryArgs()))
	return nil, nil
}

// sanitizeRedirect returns a safe local redirect target from the optional
// next/redirect query param. Only local absolute paths (starting with a single
// "/") are honored; anything else falls back to "/" to avoid open redirects.
func sanitizeRedirect(args *fasthttp.Args) string {
	next := string(args.Peek("next"))
	if next == "" {
		next = string(args.Peek("redirect"))
	}

	if len(next) >= 1 && next[0] == '/' && !strings.HasPrefix(next, "//") {
		return next
	}

	return "/"
}

func Resolve(ctx context.Context, env Env, token string) error {
	// Parse and validate JWT token.
	hotAuthToken, err := env.ParseHotAuthToken(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	// Get or create user.
	user, err := env.UserByEmail(ctx, hotAuthToken.Email)
	if err != nil {
		if db.IsNoFound(err) {
			// User doesn't exist, create new user.
			params := db.InsertUserWithEmailParams{
				Email:      hotAuthToken.Email,
				CreatedVia: "hot_auth_token",
			}
			user, err = env.InsertUserWithEmail(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get user: %w", err)
		}
	}

	// If AdminEnter flag is set, ensure user is admin.
	if hotAuthToken.AdminEnter {
		err = ensureUserIsAdmin(ctx, env, user.ID)
		if err != nil {
			return err
		}
	}

	// Create session and set cookie.
	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
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
