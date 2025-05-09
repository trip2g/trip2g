package signinbypurchasetoken

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type Env interface {
	SetupUserToken(ctx context.Context, userID int64) (string, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	PurchaseByID(ctx context.Context, id string) (db.Purchase, error)
	Logger() logger.Logger
}

const logPrefix = "signinbypurchasetoken: "

func Resolve(ctx context.Context, env Env, tokens []*model.PurchaseToken) (bool, error) {
	for _, token := range tokens {
		purchase, err := env.PurchaseByID(ctx, token.PurchaseID)
		if err != nil {
			if db.IsNoFound(err) {
				env.Logger().Info(logPrefix+"purchase not found", "token", token)
				continue
			}

			return false, fmt.Errorf("failed to find purchase by ID: %w", err)
		}

		// not yet payed
		if !purchase.UserID.Valid {
			continue
		}

		user, err := env.UserByID(ctx, purchase.UserID.Int64)
		if err != nil {
			return false, fmt.Errorf("failed to find user by email: %w", err)
		}

		_, err = env.SetupUserToken(ctx, user.ID)
		if err != nil {
			return false, fmt.Errorf("failed to setup user token: %w", err)
		}

		// will remove all tokens from the cookies
		return true, nil
	}

	return false, nil
}
