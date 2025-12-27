package deleteadmin

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg deleteadmin_test . Env

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	DeleteAdmin(ctx context.Context, userID int64) error
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
}

type Input = model.DeleteAdminInput
type Payload = model.DeleteAdminOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Prevent self-deletion
	if int64(token.ID) == input.UserID {
		return &model.ErrorPayload{Message: "Cannot delete yourself as admin"}, nil
	}

	// Check if user is an admin
	_, err = env.AdminByUserID(ctx, input.UserID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "User is not an admin"}, nil
		}
		return nil, fmt.Errorf("failed to check admin status: %w", err)
	}

	err = env.DeleteAdmin(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete admin: %w", err)
	}

	payload := model.DeleteAdminPayload{
		Success: true,
	}

	return &payload, nil
}
