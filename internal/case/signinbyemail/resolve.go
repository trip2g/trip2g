package signinbyemail

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	VerifySignInCode(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error)
	BuildUserTokenData(ctx context.Context, userID int64) (*usertoken.Data, error)
}

type Request struct {
	Email string
	Code  int64
}

type Response struct {
	tokenData *usertoken.Data

	Token string

	Errors []string
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := &Response{}

	codeParams := db.VerifySignInCodeParams{
		Email: req.Email,
		Code:  req.Code,
	}

	fmt.Printf("%+v\n", codeParams)

	userID, err := env.VerifySignInCode(ctx, codeParams)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Errors = append(response.Errors, "invalid_code")
			return response, nil
		}

		return nil, fmt.Errorf("failed to list active sign-in codes: %w", err)
	}

	// Build user token data
	tokenData, err := env.BuildUserTokenData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to build user token data: %w", err)
	}

	response.tokenData = tokenData

	return response, nil
}
