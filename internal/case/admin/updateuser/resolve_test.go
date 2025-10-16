package updateuser_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/admin/updateuser"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updateuser_test . Env

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	UserByEmail(ctx context.Context, lower string) (db.User, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.UpdateUserInput
	}

	newEmail := "updated@example.com"

	tests := []struct {
		name          string
		env           updateuser.Env
		args          args
		want          model.UpdateUserOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful update user with email",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    id,
						Email: sql.NullString{String: "old@example.com", Valid: true},
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
				UpdateUserFunc: func(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
					return db.User{
						ID:    arg.ID,
						Email: arg.Email,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID:    123,
					Email: &newEmail,
				},
			},
			want: &model.UpdateUserPayload{
				User: &db.User{
					ID:    123,
					Email: sql.NullString{String: newEmail, Valid: true},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.UpdateUserCalls(), 1)

				// Verify user ID check
				require.Equal(t, int64(123), mockEnv.UserByIDCalls()[0].ID)

				// Verify email uniqueness check
				require.Equal(t, newEmail, mockEnv.UserByEmailCalls()[0].Lower)

				// Verify update params
				updateParams := mockEnv.UpdateUserCalls()[0].Arg
				require.Equal(t, int64(123), updateParams.ID)
				require.Equal(t, newEmail, updateParams.Email.String)
				require.True(t, updateParams.Email.Valid)
			},
		},
		{
			name: "successful update user without email",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    id,
						Email: sql.NullString{String: "existing@example.com", Valid: true},
					}, nil
				},
				UpdateUserFunc: func(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
					return db.User{
						ID:    arg.ID,
						Email: sql.NullString{String: "existing@example.com", Valid: true},
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID: 123,
				},
			},
			want: &model.UpdateUserPayload{
				User: &db.User{
					ID:    123,
					Email: sql.NullString{String: "existing@example.com", Valid: true},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Len(t, mockEnv.UpdateUserCalls(), 1)

				// Verify update params
				updateParams := mockEnv.UpdateUserCalls()[0].Arg
				require.Equal(t, int64(123), updateParams.ID)
				require.False(t, updateParams.Email.Valid)
			},
		},
		{
			name: "admin token error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID: 123,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UserByIDCalls())
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.UpdateUserCalls())
			},
		},
		{
			name: "validation error - missing ID",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID: 0,
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "id", Value: "cannot be blank"}},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UserByIDCalls())
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.UpdateUserCalls())
			},
		},
		{
			name: "validation error - invalid email",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID:    123,
					Email: func() *string { e := "invalid-email"; return &e }(),
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "email", Value: "must be a valid email address"}},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UserByIDCalls())
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.UpdateUserCalls())
			},
		},
		{
			name: "user not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID: 999,
				},
			},
			want:    &model.ErrorPayload{Message: "User not found"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.UpdateUserCalls())

				require.Equal(t, int64(999), mockEnv.UserByIDCalls()[0].ID)
			},
		},
		{
			name: "user by id database error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{}, errors.New("database connection error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID: 123,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.UpdateUserCalls())
			},
		},
		{
			name: "email already exists for different user",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    id,
						Email: sql.NullString{String: "old@example.com", Valid: true},
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{
						ID:    456, // Different user ID
						Email: sql.NullString{String: lower, Valid: true},
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID:    123,
					Email: &newEmail,
				},
			},
			want:    &model.ErrorPayload{Message: "User with this email already exists"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Empty(t, mockEnv.UpdateUserCalls())

				require.Equal(t, newEmail, mockEnv.UserByEmailCalls()[0].Lower)
			},
		},
		{
			name: "email exists for same user (should succeed)",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    id,
						Email: sql.NullString{String: "existing@example.com", Valid: true},
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{
						ID:    123, // Same user ID
						Email: sql.NullString{String: lower, Valid: true},
					}, nil
				},
				UpdateUserFunc: func(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
					return db.User{
						ID:    arg.ID,
						Email: arg.Email,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID:    123,
					Email: &newEmail,
				},
			},
			want: &model.UpdateUserPayload{
				User: &db.User{
					ID:    123,
					Email: sql.NullString{String: newEmail, Valid: true},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.UpdateUserCalls(), 1)
			},
		},
		{
			name: "email check database error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    id,
						Email: sql.NullString{String: "old@example.com", Valid: true},
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, errors.New("database connection error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID:    123,
					Email: &newEmail,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Empty(t, mockEnv.UpdateUserCalls())
			},
		},
		{
			name: "update user database error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    id,
						Email: sql.NullString{String: "old@example.com", Valid: true},
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
				UpdateUserFunc: func(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
					return db.User{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID:    123,
					Email: &newEmail,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.UpdateUserCalls(), 1)
			},
		},
		{
			name: "update user unique constraint violation",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    id,
						Email: sql.NullString{String: "old@example.com", Valid: true},
					}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
				UpdateUserFunc: func(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
					return db.User{}, errors.New("UNIQUE constraint failed: users.email")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateUserInput{
					ID:    123,
					Email: &newEmail,
				},
			},
			want:    &model.ErrorPayload{Message: "User with this email already exists"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByIDCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.UpdateUserCalls(), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateuser.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Resolve() = %v, want %v", got, tt.want)
					for _, desc := range pretty.Diff(got, tt.want) {
						t.Error(desc)
					}
				}
			}

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}