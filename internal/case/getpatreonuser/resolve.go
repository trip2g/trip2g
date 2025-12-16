package getpatreonuser

import (
	"context"
	"fmt"
	"trip2g/internal/db"
)

type Env interface {
	GetPatreonMemberByEmail(ctx context.Context, email string) (db.PatreonMember, error)
	UpdatePatreonMemberUserID(ctx context.Context, args db.UpdatePatreonMemberUserIDParams) error

	UserByID(ctx context.Context, id int64) (db.User, error)
	UserByEmail(ctx context.Context, email string) (db.User, error)
	InsertUserWithEmail(ctx context.Context, params db.InsertUserWithEmailParams) (db.User, error)
}

func Resolve(ctx context.Context, env Env, email string) (*db.User, error) {
	member, err := env.GetPatreonMemberByEmail(ctx, email)
	if err != nil {
		if db.IsNoFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get patreon member by email: %w", err)
	}

	if member.UserID != nil {
		user, userErr := env.UserByID(ctx, *member.UserID)
		if userErr != nil {
			return nil, fmt.Errorf("failed to get user by ID: %w", userErr)
		}

		return &user, nil
	}

	user, userErr := env.UserByEmail(ctx, email)
	if userErr != nil {
		if db.IsNoFound(userErr) {
			params := db.InsertUserWithEmailParams{
				Email:      email,
				CreatedVia: "patreon",
			}

			user, userErr = env.InsertUserWithEmail(ctx, params)
			if userErr != nil {
				return nil, fmt.Errorf("failed to insert user with email: %w", userErr)
			}
		} else {
			return nil, fmt.Errorf("failed to get user by email: %w", userErr)
		}
	}

	updateParams := db.UpdatePatreonMemberUserIDParams{
		ID:     member.ID,
		UserID: &user.ID,
	}

	err = env.UpdatePatreonMemberUserID(ctx, updateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update patreon member user ID: %w", err)
	}

	return &user, nil
}
