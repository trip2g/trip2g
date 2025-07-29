package getpatreonuser_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/getpatreonuser"
	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg getpatreonuser_test . Env

type Env interface {
	getpatreonuser.Env
}

func TestResolve_PatreonMemberNotFound(t *testing.T) {
	// Setup
	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{}, sql.ErrNoRows
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.NoError(t, err)
	require.Nil(t, user)
	require.Len(t, env.GetPatreonMemberByEmailCalls(), 1)
}

func TestResolve_PatreonMemberWithExistingUser(t *testing.T) {
	// Setup
	expectedUser := db.User{
		ID:    123,
		Email: sql.NullString{String: "test@example.com", Valid: true},
	}

	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			require.Equal(t, "test@example.com", email)
			return db.PatreonMember{
				ID:     1,
				Email:  "test@example.com",
				UserID: sql.NullInt64{Int64: 123, Valid: true},
			}, nil
		},
		UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
			require.Equal(t, int64(123), id)
			return expectedUser, nil
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, expectedUser.ID, user.ID)
	require.Len(t, env.GetPatreonMemberByEmailCalls(), 1)
	require.Len(t, env.UserByIDCalls(), 1)
}

func TestResolve_PatreonMemberWithoutUser_UserExists(t *testing.T) {
	// Setup
	expectedUser := db.User{
		ID:    456,
		Email: sql.NullString{String: "test@example.com", Valid: true},
	}

	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{
				ID:     1,
				Email:  "test@example.com",
				UserID: sql.NullInt64{Valid: false}, // No user linked
			}, nil
		},
		UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
			require.Equal(t, "test@example.com", email)
			return expectedUser, nil
		},
		UpdatePatreonMemberUserIDFunc: func(ctx context.Context, args db.UpdatePatreonMemberUserIDParams) error {
			require.Equal(t, int64(1), args.ID)
			require.Equal(t, int64(456), args.UserID.Int64)
			require.True(t, args.UserID.Valid)
			return nil
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, expectedUser.ID, user.ID)
	require.Len(t, env.GetPatreonMemberByEmailCalls(), 1)
	require.Len(t, env.UserByEmailCalls(), 1)
	require.Len(t, env.UpdatePatreonMemberUserIDCalls(), 1)
}

func TestResolve_PatreonMemberWithoutUser_CreateNewUser(t *testing.T) {
	// Setup
	newUser := db.User{
		ID:    789,
		Email: sql.NullString{String: "test@example.com", Valid: true},
	}

	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{
				ID:     1,
				Email:  "test@example.com",
				UserID: sql.NullInt64{Valid: false}, // No user linked
			}, nil
		},
		UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, sql.ErrNoRows // User doesn't exist
		},
		InsertUserWithEmailFunc: func(ctx context.Context, email string) (db.User, error) {
			require.Equal(t, "test@example.com", email)
			return newUser, nil
		},
		UpdatePatreonMemberUserIDFunc: func(ctx context.Context, args db.UpdatePatreonMemberUserIDParams) error {
			require.Equal(t, int64(1), args.ID)
			require.Equal(t, int64(789), args.UserID.Int64)
			require.True(t, args.UserID.Valid)
			return nil
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, newUser.ID, user.ID)
	require.Len(t, env.GetPatreonMemberByEmailCalls(), 1)
	require.Len(t, env.UserByEmailCalls(), 1)
	require.Len(t, env.InsertUserWithEmailCalls(), 1)
	require.Len(t, env.UpdatePatreonMemberUserIDCalls(), 1)
}

func TestResolve_GetPatreonMemberError(t *testing.T) {
	// Setup
	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{}, errors.New("database error")
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get patreon member by email")
	require.Nil(t, user)
}

func TestResolve_UserByEmailError(t *testing.T) {
	// Setup
	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{
				ID:     1,
				Email:  "test@example.com",
				UserID: sql.NullInt64{Valid: false},
			}, nil
		},
		UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, errors.New("database error")
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get user by email")
	require.Nil(t, user)
}

func TestResolve_InsertUserError(t *testing.T) {
	// Setup
	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{
				ID:     1,
				Email:  "test@example.com",
				UserID: sql.NullInt64{Valid: false},
			}, nil
		},
		UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, sql.ErrNoRows
		},
		InsertUserWithEmailFunc: func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, errors.New("insert error")
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to insert user with email")
	require.Nil(t, user)
}

func TestResolve_UpdatePatreonMemberError(t *testing.T) {
	// Setup
	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{
				ID:     1,
				Email:  "test@example.com",
				UserID: sql.NullInt64{Valid: false},
			}, nil
		},
		UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: 123}, nil
		},
		UpdatePatreonMemberUserIDFunc: func(ctx context.Context, args db.UpdatePatreonMemberUserIDParams) error {
			return errors.New("update error")
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to update patreon member user ID")
	require.Nil(t, user)
}

func TestResolve_UserByIDError(t *testing.T) {
	// Setup
	env := &EnvMock{
		GetPatreonMemberByEmailFunc: func(ctx context.Context, email string) (db.PatreonMember, error) {
			return db.PatreonMember{
				ID:     1,
				Email:  "test@example.com",
				UserID: sql.NullInt64{Int64: 123, Valid: true},
			}, nil
		},
		UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
			return db.User{}, errors.New("user not found")
		},
	}

	// Execute
	user, err := getpatreonuser.Resolve(context.Background(), env, "test@example.com")

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get user by ID")
	require.Nil(t, user)
}