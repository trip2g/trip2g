package getboostyuser

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/ptr"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		email     string
		setupMock func(env *EnvMock)
		wantUser  *db.User
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "member not found returns nil",
			email: "notfound@example.com",
			setupMock: func(env *EnvMock) {
				env.GetBoostyMemberByEmailFunc = func(ctx context.Context, email string) (db.BoostyMember, error) {
					return db.BoostyMember{}, sql.ErrNoRows
				}
			},
			wantUser: nil,
			wantErr:  false,
		},
		{
			name:  "member with existing user_id",
			email: "existing@example.com",
			setupMock: func(env *EnvMock) {
				env.GetBoostyMemberByEmailFunc = func(ctx context.Context, email string) (db.BoostyMember, error) {
					require.Equal(t, "existing@example.com", email)
					return db.BoostyMember{
						ID:     1,
						Email:  "existing@example.com",
						UserID: ptr.To(int64(123)),
					}, nil
				}
				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					require.Equal(t, int64(123), id)
					return db.User{
						ID:    123,
						Email: ptr.To("existing@example.com"),
					}, nil
				}
			},
			wantUser: &db.User{
				ID:    123,
				Email: ptr.To("existing@example.com"),
			},
			wantErr: false,
		},
		{
			name:  "member without user_id - existing user",
			email: "nolink@example.com",
			setupMock: func(env *EnvMock) {
				env.GetBoostyMemberByEmailFunc = func(ctx context.Context, email string) (db.BoostyMember, error) {
					return db.BoostyMember{
						ID:     2,
						Email:  "nolink@example.com",
						UserID: nil,
					}, nil
				}
				env.UserByEmailFunc = func(ctx context.Context, email string) (db.User, error) {
					require.Equal(t, "nolink@example.com", email)
					return db.User{
						ID:    456,
						Email: ptr.To("nolink@example.com"),
					}, nil
				}
				env.UpdateBoostyMemberUserIDFunc = func(ctx context.Context, arg db.UpdateBoostyMemberUserIDParams) error {
					require.Equal(t, int64(2), arg.ID)
					require.NotNil(t, arg.UserID)
					require.Equal(t, int64(456), *arg.UserID)
					return nil
				}
			},
			wantUser: &db.User{
				ID:    456,
				Email: ptr.To("nolink@example.com"),
			},
			wantErr: false,
		},
		{
			name:  "member without user_id - create new user",
			email: "newuser@example.com",
			setupMock: func(env *EnvMock) {
				env.GetBoostyMemberByEmailFunc = func(ctx context.Context, email string) (db.BoostyMember, error) {
					return db.BoostyMember{
						ID:     3,
						Email:  "newuser@example.com",
						UserID: nil,
					}, nil
				}
				env.UserByEmailFunc = func(ctx context.Context, email string) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				}
				env.InsertUserWithEmailFunc = func(ctx context.Context, args db.InsertUserWithEmailParams) (db.User, error) {
					require.Equal(t, "newuser@example.com", args.Email)
					require.Equal(t, "boosty", args.CreatedVia)
					return db.User{
						ID:    789,
						Email: ptr.To("newuser@example.com"),
					}, nil
				}
				env.UpdateBoostyMemberUserIDFunc = func(ctx context.Context, arg db.UpdateBoostyMemberUserIDParams) error {
					require.Equal(t, int64(3), arg.ID)
					require.NotNil(t, arg.UserID)
					require.Equal(t, int64(789), *arg.UserID)
					return nil
				}
			},
			wantUser: &db.User{
				ID:    789,
				Email: ptr.To("newuser@example.com"),
			},
			wantErr: false,
		},
		{
			name:  "error getting member",
			email: "error@example.com",
			setupMock: func(env *EnvMock) {
				env.GetBoostyMemberByEmailFunc = func(ctx context.Context, email string) (db.BoostyMember, error) {
					return db.BoostyMember{}, errors.New("database error")
				}
			},
			wantErr: true,
			errMsg:  "failed to get boosty member by email",
		},
		{
			name:  "error updating member user_id",
			email: "updateerror@example.com",
			setupMock: func(env *EnvMock) {
				env.GetBoostyMemberByEmailFunc = func(ctx context.Context, email string) (db.BoostyMember, error) {
					return db.BoostyMember{
						ID:     4,
						Email:  "updateerror@example.com",
						UserID: nil,
					}, nil
				}
				env.UserByEmailFunc = func(ctx context.Context, email string) (db.User, error) {
					return db.User{ID: 999}, nil
				}
				env.UpdateBoostyMemberUserIDFunc = func(ctx context.Context, arg db.UpdateBoostyMemberUserIDParams) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
			errMsg:  "failed to update boosty member user ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setupMock(env)

			user, err := Resolve(ctx, env, tt.email)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				if tt.wantUser == nil {
					require.Nil(t, user)
				} else {
					require.NotNil(t, user)
					require.Equal(t, tt.wantUser.ID, user.ID)
					require.Equal(t, tt.wantUser.Email, user.Email)
				}
			}
		})
	}
}
