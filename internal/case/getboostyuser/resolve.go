package getboostyuser

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
)

type Env interface {
	GetBoostyMemberByEmail(ctx context.Context, email string) (db.BoostyMember, error)
	UpdateBoostyMemberUserID(ctx context.Context, arg db.UpdateBoostyMemberUserIDParams) error

	UserByID(ctx context.Context, id int64) (db.User, error)
	UserByEmail(ctx context.Context, email string) (db.User, error)
	InsertUserWithEmail(ctx context.Context, email string) (db.User, error)
}

func Resolve(ctx context.Context, env Env, email string) (*db.User, error) {
	member, err := env.GetBoostyMemberByEmail(ctx, email)
	if err != nil {
		if db.IsNoFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get boosty member by email: %w", err)
	}

	if member.UserID.Valid {
		user, userErr := env.UserByID(ctx, member.UserID.Int64)
		if userErr != nil {
			return nil, fmt.Errorf("failed to get user by ID: %w", userErr)
		}

		return &user, nil
	}

	user, userErr := env.UserByEmail(ctx, email)
	if userErr != nil {
		if db.IsNoFound(userErr) {
			user, userErr = env.InsertUserWithEmail(ctx, email)
			if userErr != nil {
				return nil, fmt.Errorf("failed to insert user with email: %w", userErr)
			}
		} else {
			return nil, fmt.Errorf("failed to get user by email: %w", userErr)
		}
	}

	updateParams := db.UpdateBoostyMemberUserIDParams{
		ID:     member.ID,
		UserID: sql.NullInt64{Valid: true, Int64: user.ID},
	}

	err = env.UpdateBoostyMemberUserID(ctx, updateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update boosty member user ID: %w", err)
	}

	return &user, nil
}
