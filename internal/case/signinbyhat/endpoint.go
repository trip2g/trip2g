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
		_, err = env.AdminByUserID(ctx, user.ID)
		if err != nil {
			if db.IsNoFound(err) {
				// User is not admin, make them admin.
				_, insertErr := env.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
				if insertErr != nil {
					return fmt.Errorf("failed to make user admin: %w", insertErr)
				}
			} else {
				return fmt.Errorf("failed to check admin status: %w", err)
			}
		}
	}

	// Create session and set cookie.
	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}
