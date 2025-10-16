package createuser

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	InsertUserWithEmail(ctx context.Context, arg db.InsertUserWithEmailParams) (db.User, error)
	UserByEmail(ctx context.Context, lower string) (db.User, error)
}

type Input = model.CreateUserInput
type Payload = model.CreateUserOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Email, ozzo.Required, is.Email),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Check admin authorization
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Validate input
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Check if user already exists
	_, userErr := env.UserByEmail(ctx, input.Email)
	if userErr == nil {
		return &model.ErrorPayload{Message: "User with this email already exists"}, nil
	}
	if !db.IsNoFound(userErr) {
		return nil, fmt.Errorf("failed to check existing user: %w", userErr)
	}

	// Create user
	params := db.InsertUserWithEmailParams{
		Email:      input.Email,
		CreatedVia: "admin",
	}

	user, err := env.InsertUserWithEmail(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "User with this email already exists"}, nil
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	payload := model.CreateUserPayload{
		User: &user,
	}

	return &payload, nil
}
