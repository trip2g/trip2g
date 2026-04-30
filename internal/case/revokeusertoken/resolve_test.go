package revokeusertoken_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/revokeusertoken"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve_Success(t *testing.T) {
	ctx := context.Background()

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 42, Role: "user"}, nil
		},
		RevokeUserTokenFunc: func(_ context.Context, arg db.RevokeUserTokenParams) (db.UserToken, error) {
			require.Equal(t, "token-id-1", arg.ID)
			require.Equal(t, int64(42), arg.UserID)
			return db.UserToken{ID: arg.ID, UserID: arg.UserID}, nil
		},
	}

	result, err := revokeusertoken.Resolve(ctx, env, revokeusertoken.Input{ID: "token-id-1"})
	require.NoError(t, err)

	payload, ok := result.(*revokeusertoken.SuccessPayload)
	require.True(t, ok, "expected SuccessPayload, got %T", result)
	require.Equal(t, "token-id-1", payload.Token.ID)
}

func TestResolve_NotFound(t *testing.T) {
	ctx := context.Background()

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 42, Role: "user"}, nil
		},
		RevokeUserTokenFunc: func(_ context.Context, _ db.RevokeUserTokenParams) (db.UserToken, error) {
			return db.UserToken{}, sql.ErrNoRows
		},
	}

	result, err := revokeusertoken.Resolve(ctx, env, revokeusertoken.Input{ID: "nonexistent"})
	require.NoError(t, err)

	errPayload, ok := result.(*revokeusertoken.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.NotEmpty(t, errPayload.Message)
}

func TestResolve_WrongOwner(t *testing.T) {
	ctx := context.Background()

	// RevokeUserToken uses WHERE id=? AND user_id=? so wrong owner returns sql.ErrNoRows
	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 99, Role: "user"}, nil
		},
		RevokeUserTokenFunc: func(_ context.Context, arg db.RevokeUserTokenParams) (db.UserToken, error) {
			require.Equal(t, int64(99), arg.UserID)
			return db.UserToken{}, sql.ErrNoRows
		},
	}

	result, err := revokeusertoken.Resolve(ctx, env, revokeusertoken.Input{ID: "other-user-token"})
	require.NoError(t, err)

	errPayload, ok := result.(*revokeusertoken.ErrorPayload)
	require.True(t, ok)
	require.NotEmpty(t, errPayload.Message)
}

func TestResolve_Unauthenticated(t *testing.T) {
	ctx := context.Background()

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return nil, nil
		},
		RevokeUserTokenFunc: func(_ context.Context, _ db.RevokeUserTokenParams) (db.UserToken, error) {
			t.Fatal("should not be called")
			return db.UserToken{}, nil
		},
	}

	result, err := revokeusertoken.Resolve(ctx, env, revokeusertoken.Input{ID: "some-id"})
	require.NoError(t, err)

	errPayload, ok := result.(*revokeusertoken.ErrorPayload)
	require.True(t, ok)
	require.NotEmpty(t, errPayload.Message)
}
