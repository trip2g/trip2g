package signinbyemail

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Env interface {
	VerifySignInCode(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error)
	DeleteSignInCodesByUserID(ctx context.Context, userID int64) error
	SetupUserToken(ctx context.Context, userID int64) (string, error)
}

type Request struct {
	Email string
	Code  int64
}

func (r *Request) Normalize() {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
}

func (req *Request) Validate() *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(req,
		ozzo.Field(&req.Email, ozzo.Required, is.Email),
		ozzo.Field(&req.Code, ozzo.Required, ozzo.Min(100000), ozzo.Max(999999)),
	))
}

func (req *Request) Resolve(ctx context.Context, env Env) (model.SignInOrErrorPayload, error) {
	req.Normalize()

	errorPayload := req.Validate()
	if errorPayload != nil {
		return errorPayload, nil
	}

	codeParams := db.VerifySignInCodeParams{
		Email: req.Email,
		Code:  req.Code,
	}

	userID, err := env.VerifySignInCode(ctx, codeParams)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NewFieldError("email", "not_found"), nil
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

	response := model.SignInPayload{
		Token:  token,
		Viewer: &model.Viewer{},
	}

	return &response, nil
}
