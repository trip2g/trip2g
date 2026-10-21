package signinbypassword

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"

	"golang.org/x/crypto/bcrypt"
)

type Env interface {
	OwnerEmail() string
	OwnerPasswordHash() string
	AllowPasswordSignInAttempt(key string) bool
	UserByEmail(ctx context.Context, email string) (db.User, error)
	SetupUserToken(ctx context.Context, userID int64) (string, error)
}

type Input = model.SignInByPasswordInput

// invalidCredentials is the one answer for a wrong email, a wrong password and
// an instance without a password set, so a reply never tells which it was.
func invalidCredentials() *model.ErrorPayload {
	return &model.ErrorPayload{Message: "Email or password is invalid"}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func Resolve(ctx context.Context, env Env, input Input, remoteIP string) (model.SignInOrErrorPayload, error) {
	hash := env.OwnerPasswordHash()
	if hash == "" {
		return invalidCredentials(), nil
	}

	email := normalizeEmail(input.Email)

	if !env.AllowPasswordSignInAttempt("ip:"+remoteIP) || !env.AllowPasswordSignInAttempt("email:"+email) {
		return &model.ErrorPayload{Message: "Too many attempts, try again later"}, nil
	}

	// The hash is compared whatever the email, so a wrong address takes as
	// long to refuse as a wrong password.
	passwordErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password))
	emailMatches := subtle.ConstantTimeCompare([]byte(email), []byte(normalizeEmail(env.OwnerEmail()))) == 1

	if passwordErr != nil || !emailMatches {
		return invalidCredentials(), nil //nolint:nilerr // business error as payload
	}

	user, err := env.UserByEmail(ctx, env.OwnerEmail())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner user: %w", err)
	}

	token, err := env.SetupUserToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to build user token data: %w", err)
	}

	response := model.SignInPayload{
		Token:  token,
		Viewer: &appmodel.Viewer{UserID: &user.ID},
	}

	return &response, nil
}
