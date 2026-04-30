package createusertoken

import (
	"context"
	"fmt"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/personaltoken"
	"trip2g/internal/usertoken"
)

const maxActiveTokens = 10

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	CountActiveUserTokensByUserID(ctx context.Context, userID int64) (int64, error)
	GenerateUniqID() string
	InsertUserToken(ctx context.Context, arg db.InsertUserTokenParams) (db.UserToken, error)
}

type Input struct {
	Name      string
	ExpiresIn *time.Duration
}

type SuccessPayload struct {
	PlaintextToken string
	Token          db.UserToken
}

type ErrorPayload struct {
	Message string
}

type Payload interface{ isCreateUserTokenPayload() }

func (p *SuccessPayload) isCreateUserTokenPayload() {}
func (p *ErrorPayload) isCreateUserTokenPayload()   {}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	user, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}
	if user == nil {
		return &ErrorPayload{Message: "unauthenticated"}, nil
	}

	count, err := env.CountActiveUserTokensByUserID(ctx, int64(user.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to count active user tokens: %w", err)
	}
	if count >= maxActiveTokens {
		return &ErrorPayload{Message: "token_limit_exceeded"}, nil
	}

	plaintext := personaltoken.Generate()
	hash := personaltoken.Hash(plaintext)
	prefix := personaltoken.DisplayPrefix(plaintext)

	var expiresAt *time.Time
	if input.ExpiresIn != nil {
		t := time.Now().Add(*input.ExpiresIn)
		expiresAt = &t
	}

	token, err := env.InsertUserToken(ctx, db.InsertUserTokenParams{
		ID:          env.GenerateUniqID(),
		UserID:      int64(user.ID),
		Name:        input.Name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		Scope:       "all",
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert user token: %w", err)
	}

	return &SuccessPayload{
		PlaintextToken: plaintext,
		Token:          token,
	}, nil
}
