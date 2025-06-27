package signinbytgauthtoken

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	SetupUserToken(ctx context.Context, userID int64) (string, error)
	TgUserProfileByChatIDAndBotID(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error)
	InsertUserWithTgUserID(ctx context.Context, tgUserID sql.NullInt64) (db.User, error)
	UserByTgUserID(ctx context.Context, tgUserID sql.NullInt64) (db.User, error)
	ParseTgAuthToken(ctx context.Context, token string) (*model.TgAuthToken, error)
}

var ErrProfileNotFound = errors.New("profile not found")

func Resolve(ctx context.Context, env Env, rawToken string) error {
	token, err := env.ParseTgAuthToken(ctx, rawToken)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	profileParams := db.TgUserProfileByChatIDAndBotIDParams(*token)

	// check if profile exists
	_, err = env.TgUserProfileByChatIDAndBotID(ctx, profileParams)
	if err != nil {
		if db.IsNoFound(err) {
			return ErrProfileNotFound
		}

		return fmt.Errorf("failed to get profile by chat ID and bot ID: %w", err)
	}

	sqlID := sql.NullInt64{Valid: true, Int64: token.ChatID}

	// select or create user by chat id
	user, err := env.UserByTgUserID(ctx, sqlID)
	if err != nil {
		if db.IsNoFound(err) {
			user, err = env.InsertUserWithTgUserID(ctx, sqlID)
			if err != nil {
				return fmt.Errorf("failed to insert user with TG user ID: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get user by TG user ID: %w", err)
		}
	}

	_, err = env.SetupUserToken(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to setup user token: %w", err)
	}

	return nil
}
