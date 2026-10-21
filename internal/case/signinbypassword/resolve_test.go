package signinbypassword

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg signinbypassword . Env

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const ownerEmail = "Owner@Example.com"

func ownerHash(t *testing.T) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.MinCost)
	require.NoError(t, err)

	return string(hash)
}

func newEnv(hash string) *EnvMock {
	return &EnvMock{
		OwnerEmailFunc:                 func() string { return ownerEmail },
		OwnerPasswordHashFunc:          func() string { return hash },
		AllowPasswordSignInAttemptFunc: func(string) bool { return true },
		UserByEmailFunc: func(_ context.Context, email string) (db.User, error) {
			return db.User{ID: 1, Email: &email}, nil
		},
		SetupUserTokenFunc: func(context.Context, int64) (string, error) {
			return "tok", nil
		},
	}
}

func TestResolve(t *testing.T) {
	hash := ownerHash(t)
	userID := int64(1)

	tests := []struct {
		name  string
		hash  string
		input Input
		want  model.SignInOrErrorPayload
	}{
		{
			name:  "owner with the right password",
			hash:  hash,
			input: Input{Email: "owner@example.com", Password: "correct horse"},
			want:  &model.SignInPayload{Token: "tok", Viewer: &appmodel.Viewer{UserID: &userID}},
		},
		{
			name:  "email matched case-insensitively and trimmed",
			hash:  hash,
			input: Input{Email: "  OWNER@example.COM ", Password: "correct horse"},
			want:  &model.SignInPayload{Token: "tok", Viewer: &appmodel.Viewer{UserID: &userID}},
		},
		{
			name:  "wrong password",
			hash:  hash,
			input: Input{Email: "owner@example.com", Password: "wrong"},
			want:  invalidCredentials(),
		},
		{
			name:  "another email with the owner's password",
			hash:  hash,
			input: Input{Email: "reader@example.com", Password: "correct horse"},
			want:  invalidCredentials(),
		},
		{
			name:  "feature off",
			hash:  "",
			input: Input{Email: "owner@example.com", Password: "correct horse"},
			want:  invalidCredentials(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(tt.hash)

			got, err := Resolve(context.Background(), env, tt.input, "203.0.113.7")
			require.NoError(t, err)

			if diff := pretty.Diff(got, tt.want); len(diff) > 0 {
				t.Errorf("Resolve() mismatch: %v", diff)
			}

			if _, ok := tt.want.(*model.SignInPayload); !ok {
				require.Empty(t, env.SetupUserTokenCalls(), "a refused sign-in must not mint a session")
			}
		})
	}
}

func TestResolve_LooksUpTheOwnerByConfiguredEmail(t *testing.T) {
	env := newEnv(ownerHash(t))

	_, err := Resolve(context.Background(), env, Input{Email: "owner@example.com", Password: "correct horse"}, "203.0.113.7")
	require.NoError(t, err)

	calls := env.UserByEmailCalls()
	require.Len(t, calls, 1)
	require.Equal(t, ownerEmail, calls[0].Email)
}

func TestResolve_RateLimited(t *testing.T) {
	tests := []struct {
		name    string
		blocked string
	}{
		{name: "by ip", blocked: "ip:203.0.113.7"},
		{name: "by email", blocked: "email:owner@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(ownerHash(t))
			env.AllowPasswordSignInAttemptFunc = func(key string) bool {
				return key != tt.blocked
			}

			got, err := Resolve(context.Background(), env, Input{Email: "owner@example.com", Password: "correct horse"}, "203.0.113.7")
			require.NoError(t, err)

			payload, ok := got.(*model.ErrorPayload)
			require.True(t, ok, "expected *model.ErrorPayload, got %T", got)
			require.Equal(t, "Too many attempts, try again later", payload.Message)
			require.Empty(t, env.SetupUserTokenCalls())
		})
	}
}

func TestResolve_SystemErrorPropagated(t *testing.T) {
	env := newEnv(ownerHash(t))
	env.SetupUserTokenFunc = func(context.Context, int64) (string, error) {
		return "", errors.New("store failed")
	}

	got, err := Resolve(context.Background(), env, Input{Email: "owner@example.com", Password: "correct horse"}, "203.0.113.7")
	require.Error(t, err)
	require.Nil(t, got)
}
