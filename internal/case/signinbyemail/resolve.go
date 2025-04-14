package signinbyemail

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trip2g/internal/db"
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
