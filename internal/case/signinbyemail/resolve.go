package signinbyemail

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/validator"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	VerifySignInCode(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error)
	DeleteSignInCodesByUserID(ctx context.Context, userID int64) error
	SetupUserToken(ctx context.Context, userID int64) (string, error)
}

type Request struct {
	Email string
	Code  int64
}

var ErrInvalidEmail = errors.New("invalid email")
var ErrInvalidCode = errors.New("invalid code")

func (r *Request) Normalize() {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
}

func (r *Request) Validate() error {
	err := validator.CheckEmail(r.Email)
	if err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}

	if r.Code < 100000 || r.Code > 999999 {
		return ErrInvalidCode
	}

	return nil
}

type Response struct {
	Token string

	Errors []string
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := &Response{}

	codeParams := db.VerifySignInCodeParams{
		Email: req.Email,
		Code:  req.Code,
	}

	userID, err := env.VerifySignInCode(ctx, codeParams)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.Errors = append(response.Errors, "invalid_code")
			return response, nil
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

	response.Token = token

	return response, nil
}
