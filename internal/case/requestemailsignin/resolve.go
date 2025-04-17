package requestemailsignin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"trip2g/internal/apperrors"
	"trip2g/internal/db"
	"trip2g/internal/validator"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	QueueRequestSignInEmail(ctx context.Context, email string, code int64) error
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	CountActiveSignInCodes(ctx context.Context, userID int64) (int64, error)
	CreateSignInCode(ctx context.Context, userID int64) (int64, error)
}

type Request struct {
	Email string
}

func (r *Request) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

func (r *Request) Validate() error {
	err := validator.CheckEmail(r.Email)
	if err != nil {
		return &apperrors.JSONError{Message: "invalid email"}
	}

	return nil
}

type Response struct {
	Success bool
	Errors  []string
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := &Response{}

	user, err := env.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.Errors = append(response.Errors, "user_not_found")
			return response, nil
		}

		return nil, err
	}

	count, err := env.CountActiveSignInCodes(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active signin codes: %w", err)
	}

	if count > 3 {
		response.Errors = append(response.Errors, "too_many_sign_in_codes")
		return response, nil
	}

	code, err := env.CreateSignInCode(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create signin code: %w", err)
	}

	err = env.QueueRequestSignInEmail(ctx, req.Email, code)
	if err != nil {
		return nil, fmt.Errorf("failed to queue signin email: %w", err)
	}

	response.Success = true

	return response, nil
}
