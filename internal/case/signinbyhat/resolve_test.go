package signinbyhat_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"trip2g/internal/case/signinbyhat"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type mockEnv struct {
	hotAuthToken *model.HotAuthToken
	parseErr     error

	user    db.User
	userErr error

	createUser    db.User
	createUserErr error

	admin    db.Admin
	adminErr error

	createAdmin    db.Admin
	createAdminErr error

	setupTokenErr error
}

func (m *mockEnv) ParseHotAuthToken(_ context.Context, _ string) (*model.HotAuthToken, error) {
	return m.hotAuthToken, m.parseErr
}

func (m *mockEnv) UserByEmail(_ context.Context, _ string) (db.User, error) {
	return m.user, m.userErr
}

func (m *mockEnv) InsertUserWithEmail(_ context.Context, params db.InsertUserWithEmailParams) (db.User, error) {
	if m.createUserErr != nil {
		return db.User{}, m.createUserErr
	}
	m.createUser.Email = &params.Email
	return m.createUser, nil
}

func (m *mockEnv) AdminByUserID(_ context.Context, _ int64) (db.Admin, error) {
	return m.admin, m.adminErr
}

func (m *mockEnv) InsertAdmin(_ context.Context, params db.InsertAdminParams) (db.Admin, error) {
	if m.createAdminErr != nil {
		return db.Admin{}, m.createAdminErr
	}
	m.createAdmin.UserID = params.UserID
	return m.createAdmin, nil
}

func (m *mockEnv) SetupUserToken(_ context.Context, _ int64) (string, error) {
	return "mock-session-token", m.setupTokenErr
}

func TestResolve_InvalidToken(t *testing.T) {
	env := &mockEnv{
		parseErr: errors.New("invalid signature"),
	}

	err := signinbyhat.Resolve(context.Background(), env, "bad-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse token")
}

func TestResolve_ExistingUser_NotAdmin(t *testing.T) {
	email := "user@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      email,
			AdminEnter: false,
		},
		user:    db.User{ID: 123, Email: &email},
		userErr: nil,
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.NoError(t, err)
}

func TestResolve_ExistingUser_AlreadyAdmin(t *testing.T) {
	email := "admin@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      email,
			AdminEnter: true,
		},
		user:  db.User{ID: 456, Email: &email},
		admin: db.Admin{UserID: 456},
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.NoError(t, err)
}

func TestResolve_NewUser_NotAdmin(t *testing.T) {
	email := "newuser@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      email,
			AdminEnter: false,
		},
		userErr:    sql.ErrNoRows,
		createUser: db.User{ID: 999, Email: &email},
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.NoError(t, err)
	require.Equal(t, email, *env.createUser.Email)
}

func TestResolve_NewUser_MakeAdmin(t *testing.T) {
	email := "newadmin@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      email,
			AdminEnter: true,
		},
		userErr:     sql.ErrNoRows,
		createUser:  db.User{ID: 111, Email: &email},
		adminErr:    sql.ErrNoRows,
		createAdmin: db.Admin{UserID: 111},
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.NoError(t, err)
	require.Equal(t, email, *env.createUser.Email)
	require.Equal(t, int64(111), env.createAdmin.UserID)
}

func TestResolve_ExistingUser_UpgradeToAdmin(t *testing.T) {
	email := "user@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      email,
			AdminEnter: true,
		},
		user:        db.User{ID: 333, Email: &email},
		adminErr:    sql.ErrNoRows,
		createAdmin: db.Admin{UserID: 333},
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.NoError(t, err)
	require.Equal(t, int64(333), env.createAdmin.UserID)
}

func TestResolve_CreateUserFails(t *testing.T) {
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      "newuser@example.com",
			AdminEnter: false,
		},
		userErr:       sql.ErrNoRows,
		createUserErr: errors.New("database error"),
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create user")
}

func TestResolve_MakeAdminFails(t *testing.T) {
	email := "user@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      email,
			AdminEnter: true,
		},
		user:           db.User{ID: 555, Email: &email},
		adminErr:       sql.ErrNoRows,
		createAdminErr: errors.New("admin insert failed"),
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to make user admin")
}

func TestResolve_SetupTokenFails(t *testing.T) {
	email := "user@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{
			Email:      email,
			AdminEnter: false,
		},
		user:          db.User{ID: 666, Email: &email},
		setupTokenErr: errors.New("cookie error"),
	}

	err := signinbyhat.Resolve(context.Background(), env, "valid-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create session")
}
