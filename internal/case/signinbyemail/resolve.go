package signinbyemail

import (
	"context"
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

// keeping database/sql import for sql.ErrNoRows check

type Env interface {
	VerifySignInCode(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error)
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

	codeParams := db.VerifySignInCodeParams{
		Email: &req.Email,
		Code:  req.Code,
	}

	userID, err := env.VerifySignInCode(ctx, codeParams)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gmodel.NewFieldError("code", "Code is invalid or expired"), nil
		}

		return nil, fmt.Errorf("failed to list active sign-in codes: %w", err)
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
