package createadmin

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg createadmin_test . Env

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	InsertAdmin(ctx context.Context, arg db.InsertAdminParams) (db.Admin, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
}

type Input = model.CreateAdminInput
type Payload = model.CreateAdminOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Check if user exists
	_, err = env.UserByID(ctx, input.UserID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "User not found"}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is already an admin
	_, err = env.AdminByUserID(ctx, input.UserID)
	if err == nil {
		return &model.ErrorPayload{Message: "User is already an admin"}, nil
	}
	if !db.IsNoFound(err) {
		return nil, fmt.Errorf("failed to check admin status: %w", err)
	}

	params := db.InsertAdminParams{
		UserID:    input.UserID,
		GrantedBy: ptr.To(int64(token.ID)),
	}

	admin, err := env.InsertAdmin(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "User is already an admin"}, nil
		}
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	payload := model.CreateAdminPayload{
		Admin: &admin,
	}

	return &payload, nil
}
