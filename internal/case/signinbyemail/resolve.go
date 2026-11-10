package signinbyemail

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"trip2g/internal/db"
	gmodel "trip2g/internal/graph/model"
	"trip2g/internal/model"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Wrong guesses allowed per code before all of the user's codes are dropped.
const maxCodeAttempts = 5

type Env interface {
	EmailSignInEnabled(ctx context.Context) bool
	SpendSignInCodeAttempt(ctx context.Context, email *string) ([]db.SpendSignInCodeAttemptRow, error)
	DeleteSignInCodesByUserID(ctx context.Context, userID int64) error
	SetupUserToken(ctx context.Context, userID int64) (string, error)
	DevSignInBypass(code string) bool
	UserByEmail(ctx context.Context, email string) (db.User, error)
}

func normalizeRequest(r *gmodel.SignInByEmailInput) {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
}

func validateRequest(r *gmodel.SignInByEmailInput) *gmodel.ErrorPayload {
	return gmodel.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Email, ozzo.Required, is.Email),
		ozzo.Field(&r.Code, ozzo.Required, ozzo.Length(6, 6)),
	))
}

func Resolve(ctx context.Context, env Env, req gmodel.SignInByEmailInput) (gmodel.SignInOrErrorPayload, error) {
	if !env.EmailSignInEnabled(ctx) {
		return &gmodel.ErrorPayload{Message: "email_sign_in_disabled"}, nil
	}

	normalizeRequest(&req)

	errorPayload := validateRequest(&req)
	if errorPayload != nil {
		return errorPayload, nil
	}

	if env.DevSignInBypass(req.Code) {
		user, err := env.UserByEmail(ctx, req.Email)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return gmodel.NewFieldError("code", "Code is invalid or expired"), nil
			}
			return nil, fmt.Errorf("failed to resolve user by email: %w", err)
		}
		token, err := env.SetupUserToken(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to build user token data: %w", err)
		}
		return &gmodel.SignInPayload{
			Token:  token,
			Viewer: &model.Viewer{UserID: &user.ID},
		}, nil
	}

	rows, err := env.SpendSignInCodeAttempt(ctx, &req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sign-in codes: %w", err)
	}

	var userID int64
	matched, exhausted := false, false
	for _, r := range rows {
		userID = r.UserID
		// failed_attempts already includes this attempt.
		if r.FailedAttempts >= maxCodeAttempts {
			exhausted = true
		}
		if r.FailedAttempts <= maxCodeAttempts && subtle.ConstantTimeCompare([]byte(r.Code), []byte(req.Code)) == 1 {
			matched = true
		}
	}

	if !matched {
		if exhausted {
			err = env.DeleteSignInCodesByUserID(ctx, userID)
			if err != nil {
				return nil, fmt.Errorf("failed to delete exhausted sign-in codes: %w", err)
			}
		}

		return gmodel.NewFieldError("code", "Code is invalid or expired"), nil
	}

	token, err := env.SetupUserToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to build user token data: %w", err)
	}

	err = env.DeleteSignInCodesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete sign-in codes: %w", err)
	}

	response := gmodel.SignInPayload{
		Token: token,
		Viewer: &model.Viewer{
			UserID: &userID,
		},
	}

	return &response, nil
}
