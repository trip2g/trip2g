package revokeusertoken

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	RevokeUserToken(ctx context.Context, arg db.RevokeUserTokenParams) (db.UserToken, error)
}

type Input struct {
	ID string
}

type SuccessPayload struct {
	Token db.UserToken
}

type ErrorPayload struct {
	Message string
}

type Payload interface{ isRevokeUserTokenPayload() }

func (p *SuccessPayload) isRevokeUserTokenPayload() {}
func (p *ErrorPayload) isRevokeUserTokenPayload()   {}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	user, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}
	if user == nil {
		return &ErrorPayload{Message: "unauthenticated"}, nil
	}

	token, err := env.RevokeUserToken(ctx, db.RevokeUserTokenParams{
		ID:     input.ID,
		UserID: int64(user.ID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &ErrorPayload{Message: "not_found"}, nil
		}
		return nil, fmt.Errorf("failed to revoke user token: %w", err)
	}

	return &SuccessPayload{Token: token}, nil
}
