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
	"trip2g/internal/model"

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
			name: "no active codes for the email returns code field error",
			env: &EnvMock{
				EmailSignInEnabledFunc: func(context.Context) bool { return true },
				DevSignInBypassFunc:    func(string) bool { return false },
				SpendSignInCodeAttemptFunc: func(context.Context, *string) ([]db.SpendSignInCodeAttemptRow, error) {
					return nil, nil
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
				EmailSignInEnabledFunc: func(context.Context) bool { return true },
				DevSignInBypassFunc:    func(string) bool { return false },
				SpendSignInCodeAttemptFunc: func(context.Context, *string) ([]db.SpendSignInCodeAttemptRow, error) {
					return nil, errors.New("db connection lost")
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
				EmailSignInEnabledFunc: func(context.Context) bool { return true },
				DevSignInBypassFunc:    func(string) bool { return true },
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
// with the expected token is returned and neither SpendSignInCodeAttempt nor
// DeleteSignInCodesByUserID is ever called.
func TestDevBypass_Success(t *testing.T) {
	mock := &EnvMock{
		EmailSignInEnabledFunc: func(context.Context) bool { return true },
		DevSignInBypassFunc:    func(string) bool { return true },
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

	require.Empty(t, mock.SpendSignInCodeAttemptCalls(), "SpendSignInCodeAttempt must not be called in dev bypass")
	require.Empty(t, mock.DeleteSignInCodesByUserIDCalls(), "DeleteSignInCodesByUserID must not be called in dev bypass")
}

// A disabled instance must refuse the sign-in itself, not just hide the form:
// the dev bypass is checked after the switch, so a held code is worth nothing.
func TestEmailSignInDisabled_Refuses(t *testing.T) {
	mock := &EnvMock{
		EmailSignInEnabledFunc: func(context.Context) bool { return false },
	}

	result, err := Resolve(context.Background(), mock, gmodel.SignInByEmailInput{
		Email: "hello@example.com",
		Code:  "111111",
	})
	require.NoError(t, err)

	errPayload, ok := result.(*gmodel.ErrorPayload)
	require.True(t, ok, "expected *gmodel.ErrorPayload, got %T", result)
	require.Equal(t, "email_sign_in_disabled", errPayload.Message)
	require.Empty(t, mock.SpendSignInCodeAttemptCalls(), "must not verify a code")
	require.Empty(t, mock.SetupUserTokenCalls(), "must not mint a token")
}

func TestResolve_AttemptLimit(t *testing.T) {
	invalid := &gmodel.ErrorPayload{
		ByFields: []gmodel.FieldMessage{
			{Name: "code", Value: "Code is invalid or expired"},
		},
	}
	userID := int64(7)
	signedIn := &gmodel.SignInPayload{
		Token:  "tok",
		Viewer: &model.Viewer{UserID: &userID},
	}

	tests := []struct {
		name        string
		rows        []db.SpendSignInCodeAttemptRow
		want        gmodel.SignInOrErrorPayload
		wantDeleted []int64
	}{
		{
			name: "first wrong guess is charged and the codes stay",
			rows: []db.SpendSignInCodeAttemptRow{{UserID: 7, Code: "654321", FailedAttempts: 1}},
			want: invalid,
		},
		{
			name: "fifth wrong guess deletes the codes",
			rows: []db.SpendSignInCodeAttemptRow{
				{UserID: 7, Code: "654321", FailedAttempts: 5},
				{UserID: 7, Code: "111222", FailedAttempts: 2},
			},
			want:        invalid,
			wantDeleted: []int64{7},
		},
		{
			name:        "correct code after the limit fails",
			rows:        []db.SpendSignInCodeAttemptRow{{UserID: 7, Code: "123456", FailedAttempts: 6}},
			want:        invalid,
			wantDeleted: []int64{7},
		},
		{
			name:        "correct code on the last allowed attempt succeeds",
			rows:        []db.SpendSignInCodeAttemptRow{{UserID: 7, Code: "123456", FailedAttempts: 5}},
			want:        signedIn,
			wantDeleted: []int64{7},
		},
		{
			name: "correct code among several active codes succeeds",
			rows: []db.SpendSignInCodeAttemptRow{
				{UserID: 7, Code: "654321", FailedAttempts: 3},
				{UserID: 7, Code: "123456", FailedAttempts: 1},
			},
			want:        signedIn,
			wantDeleted: []int64{7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				EmailSignInEnabledFunc: func(context.Context) bool { return true },
				DevSignInBypassFunc:    func(string) bool { return false },
				SpendSignInCodeAttemptFunc: func(context.Context, *string) ([]db.SpendSignInCodeAttemptRow, error) {
					return tt.rows, nil
				},
				DeleteSignInCodesByUserIDFunc: func(context.Context, int64) error { return nil },
				SetupUserTokenFunc: func(context.Context, int64) (string, error) {
					return "tok", nil
				},
			}

			got, err := Resolve(context.Background(), env, gmodel.SignInByEmailInput{
				Email: "User@Example.com ",
				Code:  "123456",
			})
			require.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve() mismatch")
				for _, desc := range pretty.Diff(got, tt.want) {
					t.Error(desc)
				}
			}

			require.Len(t, env.SpendSignInCodeAttemptCalls(), 1)
			require.Equal(t, "user@example.com", *env.SpendSignInCodeAttemptCalls()[0].Email)

			var deleted []int64
			for _, c := range env.DeleteSignInCodesByUserIDCalls() {
				deleted = append(deleted, c.UserID)
			}
			require.Equal(t, tt.wantDeleted, deleted)
		})
	}
}
