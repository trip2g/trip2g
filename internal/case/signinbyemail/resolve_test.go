package signinbyemail

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg signinbyemail . Env

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/db"
	gmodel "trip2g/internal/graph/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		env     Env
		req     gmodel.SignInByEmailInput
		want    gmodel.SignInOrErrorPayload
		wantErr bool
	}{
		{
			name: "invalid code - sql.ErrNoRows returns code field error",
			env: &EnvMock{
				DevSignInBypassFunc: func(string) bool { return false },
				VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
					return 0, sql.ErrNoRows
				},
			},
			req: gmodel.SignInByEmailInput{
				Email: "user@example.com",
				Code:  "123456",
			},
			want: &gmodel.ErrorPayload{
				ByFields: []gmodel.FieldMessage{
					{Name: "code", Value: "Code is invalid or expired"},
				},
			},
			wantErr: false,
		},
		{
			name: "system error propagated",
			env: &EnvMock{
				DevSignInBypassFunc: func(string) bool { return false },
				VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
					return 0, errors.New("db connection lost")
				},
			},
			req: gmodel.SignInByEmailInput{
				Email: "user@example.com",
				Code:  "123456",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "dev bypass - UserByEmail returns sql.ErrNoRows → code field error",
			env: &EnvMock{
				DevSignInBypassFunc: func(string) bool { return true },
				UserByEmailFunc: func(_ context.Context, _ string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
			},
			req: gmodel.SignInByEmailInput{
				Email: "unknown@example.com",
				Code:  "111111",
			},
			want: &gmodel.ErrorPayload{
				ByFields: []gmodel.FieldMessage{
					{Name: "code", Value: "Code is invalid or expired"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(context.Background(), tt.env, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve() mismatch")
				for _, desc := range pretty.Diff(got, tt.want) {
					t.Error(desc)
				}
			}
		})
	}
}

// TestDevBypass_Success asserts that when the dev bypass fires, a SignInPayload
// with the expected token is returned and neither VerifySignInCode nor
// DeleteSignInCodesByUserID is ever called.
func TestDevBypass_Success(t *testing.T) {
	mock := &EnvMock{
		DevSignInBypassFunc: func(string) bool { return true },
		UserByEmailFunc: func(_ context.Context, _ string) (db.User, error) {
			return db.User{ID: 7}, nil
		},
		SetupUserTokenFunc: func(_ context.Context, _ int64) (string, error) {
			return "dev-tok", nil
		},
	}

	result, err := Resolve(context.Background(), mock, gmodel.SignInByEmailInput{
		Email: "hello@example.com",
		Code:  "111111",
	})
	require.NoError(t, err)

	payload, ok := result.(*gmodel.SignInPayload)
	require.True(t, ok, "expected *gmodel.SignInPayload, got %T", result)
	require.Equal(t, "dev-tok", payload.Token)
	require.NotNil(t, payload.Viewer)
	require.Equal(t, int64(7), *payload.Viewer.UserID)

	require.Len(t, mock.VerifySignInCodeCalls(), 0, "VerifySignInCode must not be called in dev bypass")
	require.Len(t, mock.DeleteSignInCodesByUserIDCalls(), 0, "DeleteSignInCodesByUserID must not be called in dev bypass")
}
