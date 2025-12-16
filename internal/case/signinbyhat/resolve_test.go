package signinbyhat_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/signinbyhat"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/ptr"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg signinbyhat_test . Env

type Env interface {
	ParseHotAuthToken(_ context.Context, token string) (*model.HotAuthToken, error)
	SetupUserToken(ctx context.Context, userID int64) (string, error)
	UserByEmail(ctx context.Context, email string) (db.User, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx      context.Context
		rawToken string
	}

	tests := []struct {
		name          string
		env           signinbyhat.Env
		args          args
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful sign in with hot auth token",
			env: &envMock{
				ParseHotAuthTokenFunc: func(ctx context.Context, token string) (*model.HotAuthToken, error) {
					return &model.HotAuthToken{
						Email: "user@example.com",
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
					return db.User{
						ID:    123,
						Email: ptr.To("user@example.com"),
					}, nil
				},
				SetupUserTokenFunc: func(ctx context.Context, userID int64) (string, error) {
					return "new-user-token", nil
				},
			},
			args: args{
				ctx:      context.Background(),
				rawToken: "valid-hot-auth-token",
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.ParseHotAuthTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.SetupUserTokenCalls(), 1)

				// Verify token parsing
				require.Equal(t, "valid-hot-auth-token", mockEnv.ParseHotAuthTokenCalls()[0].Token)

				// Verify user lookup
				require.Equal(t, "user@example.com", mockEnv.UserByEmailCalls()[0].Email)

				// Verify token setup
				require.Equal(t, int64(123), mockEnv.SetupUserTokenCalls()[0].UserID)
			},
		},
		{
			name: "error - invalid hot auth token",
			env: &envMock{
				ParseHotAuthTokenFunc: func(ctx context.Context, token string) (*model.HotAuthToken, error) {
					return nil, errors.New("invalid token")
				},
			},
			args: args{
				ctx:      context.Background(),
				rawToken: "invalid-token",
			},
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.ParseHotAuthTokenCalls(), 1)
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.SetupUserTokenCalls())
			},
		},
		{
			name: "error - user not found",
			env: &envMock{
				ParseHotAuthTokenFunc: func(ctx context.Context, token string) (*model.HotAuthToken, error) {
					return &model.HotAuthToken{
						Email: "nonexistent@example.com",
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
			},
			args: args{
				ctx:      context.Background(),
				rawToken: "valid-token-for-missing-user",
			},
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.ParseHotAuthTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Empty(t, mockEnv.SetupUserTokenCalls())
			},
		},
		{
			name: "error - token setup fails",
			env: &envMock{
				ParseHotAuthTokenFunc: func(ctx context.Context, token string) (*model.HotAuthToken, error) {
					return &model.HotAuthToken{
						Email: "user@example.com",
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
					return db.User{
						ID:    456,
						Email: ptr.To("user@example.com"),
					}, nil
				},
				SetupUserTokenFunc: func(ctx context.Context, userID int64) (string, error) {
					return "", errors.New("token service unavailable")
				},
			},
			args: args{
				ctx:      context.Background(),
				rawToken: "valid-hot-auth-token",
			},
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.ParseHotAuthTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.SetupUserTokenCalls(), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := signinbyhat.Resolve(tt.args.ctx, tt.env, tt.args.rawToken)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}
