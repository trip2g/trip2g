package createuser_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/admin/createuser"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createuser_test . Env

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	InsertUserWithEmail(ctx context.Context, arg db.InsertUserWithEmailParams) (db.User, error)
	UserByEmail(ctx context.Context, lower string) (db.User, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateUserInput
	}

	tests := []struct {
		name          string
		env           createuser.Env
		args          args
		want          model.CreateUserOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful create user",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
				InsertUserWithEmailFunc: func(ctx context.Context, arg db.InsertUserWithEmailParams) (db.User, error) {
					return db.User{
						ID:         123,
						Email:      sql.NullString{String: arg.Email, Valid: true},
						CreatedVia: arg.CreatedVia,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateUserInput{
					Email: "test@example.com",
				},
			},
			want: &model.CreateUserPayload{
				User: &db.User{
					ID:         123,
					Email:      sql.NullString{String: "test@example.com", Valid: true},
					CreatedVia: "admin",
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.InsertUserWithEmailCalls(), 1)

				// Verify email check
				require.Equal(t, "test@example.com", mockEnv.UserByEmailCalls()[0].Lower)

				// Verify user creation params
				userParams := mockEnv.InsertUserWithEmailCalls()[0].Arg
				require.Equal(t, "test@example.com", userParams.Email)
				require.Equal(t, "admin", userParams.CreatedVia)
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
				input: model.CreateUserInput{
					Email: "test@example.com",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.InsertUserWithEmailCalls())
			},
		},
		{
			name: "validation error - empty email",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateUserInput{
					Email: "",
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "email", Value: "cannot be blank"}},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.InsertUserWithEmailCalls())
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
				input: model.CreateUserInput{
					Email: "invalid-email",
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{{Name: "email", Value: "must be a valid email address"}},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UserByEmailCalls())
				require.Empty(t, mockEnv.InsertUserWithEmailCalls())
			},
		},
		{
			name: "user already exists",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{
						ID:    456,
						Email: sql.NullString{String: lower, Valid: true},
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateUserInput{
					Email: "existing@example.com",
				},
			},
			want:    &model.ErrorPayload{Message: "User with this email already exists"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Empty(t, mockEnv.InsertUserWithEmailCalls())

				require.Equal(t, "existing@example.com", mockEnv.UserByEmailCalls()[0].Lower)
			},
		},
		{
			name: "user check database error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, errors.New("database connection error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateUserInput{
					Email: "test@example.com",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Empty(t, mockEnv.InsertUserWithEmailCalls())
			},
		},
		{
			name: "insert user database error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
				InsertUserWithEmailFunc: func(ctx context.Context, arg db.InsertUserWithEmailParams) (db.User, error) {
					return db.User{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateUserInput{
					Email: "test@example.com",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.InsertUserWithEmailCalls(), 1)
			},
		},
		{
			name: "insert user unique constraint violation",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByEmailFunc: func(ctx context.Context, lower string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
				InsertUserWithEmailFunc: func(ctx context.Context, arg db.InsertUserWithEmailParams) (db.User, error) {
					return db.User{}, errors.New("UNIQUE constraint failed: users.email")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateUserInput{
					Email: "test@example.com",
				},
			},
			want:    &model.ErrorPayload{Message: "User with this email already exists"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UserByEmailCalls(), 1)
				require.Len(t, mockEnv.InsertUserWithEmailCalls(), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createuser.Resolve(tt.args.ctx, tt.env, tt.args.input)
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
