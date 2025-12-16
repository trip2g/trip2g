package updateuser

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
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	UserByEmail(ctx context.Context, lower string) (db.User, error)
}

type Input = model.UpdateUserInput
type Payload = model.UpdateUserOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.ID, ozzo.Required),
		ozzo.Field(&r.Email, ozzo.When(r.Email != nil, is.Email)),
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

	// Check if user exists
	_, userErr := env.UserByID(ctx, input.ID)
	if userErr != nil {
		if db.IsNoFound(userErr) {
			return &model.ErrorPayload{Message: "User not found"}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", userErr)
	}

	// Check email uniqueness if email is being updated
	if input.Email != nil {
		existingUser, emailErr := env.UserByEmail(ctx, *input.Email)
		if emailErr == nil && existingUser.ID != input.ID {
			return &model.ErrorPayload{Message: "User with this email already exists"}, nil
		}
		if emailErr != nil && !db.IsNoFound(emailErr) {
			return nil, fmt.Errorf("failed to check email uniqueness: %w", emailErr)
		}
	}

	// Update user
	params := db.UpdateUserParams{
		ID: input.ID,
	}

	if input.Email != nil {
		params.Email = input.Email
	}

	updatedUser, err := env.UpdateUser(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "User with this email already exists"}, nil
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	payload := model.UpdateUserPayload{
		User: &updatedUser,
	}

	return &payload, nil
}
