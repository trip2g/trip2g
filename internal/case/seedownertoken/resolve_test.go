package seedownertoken_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/seedownertoken"
	"trip2g/internal/db"
	"trip2g/internal/personaltoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg seedownertoken_test . Env

const value = "t2g_seedvalue"

func baseEnv() *EnvMock {
	return &EnvMock{
		UserByEmailFunc: func(_ context.Context, _ string) (db.User, error) {
			return db.User{ID: 7}, nil
		},
		UserTokenByHashAnyFunc: func(_ context.Context, _ string) (db.UserToken, error) {
			return db.UserToken{}, sql.ErrNoRows
		},
		InsertUserTokenFunc: func(_ context.Context, arg db.InsertUserTokenParams) (db.UserToken, error) {
			return db.UserToken{ID: arg.ID, UserID: arg.UserID, Name: arg.Name, TokenHash: arg.TokenHash}, nil
		},
		RevokeSupersededOwnerTokensFunc: func(_ context.Context, _ db.RevokeSupersededOwnerTokensParams) (int64, error) {
			return 0, nil
		},
		GenerateUniqIDFunc: func() string { return "row-1" },
	}
}

// An absent row is the first boot on a new value: the seeder inserts it for the
// owner, hashed, under the reserved name.
func TestSeedsWhenTheRowIsAbsent(t *testing.T) {
	env := baseEnv()

	res, err := seedownertoken.Resolve(context.Background(), env, "owner@example.com", value)
	require.NoError(t, err)
	require.Equal(t, seedownertoken.OutcomeSeeded, res.Outcome)

	require.Len(t, env.InsertUserTokenCalls(), 1)
	arg := env.InsertUserTokenCalls()[0].Arg
	require.Equal(t, "row-1", arg.ID)
	require.Equal(t, int64(7), arg.UserID)
	require.Equal(t, personaltoken.ReservedOwnerTokenName, arg.Name)
	require.Equal(t, personaltoken.Hash(value), arg.TokenHash)
	require.Equal(t, personaltoken.DisplayPrefix(value), arg.TokenPrefix)
	require.Equal(t, "all", arg.Scope)
	require.Nil(t, arg.ExpiresAt, "the seeded token must not expire under the agent")
}

// The common path — unchanged value, live row — writes nothing.
func TestFindsTheLiveRowAndInsertsNothing(t *testing.T) {
	env := baseEnv()
	env.UserTokenByHashAnyFunc = func(_ context.Context, hash string) (db.UserToken, error) {
		require.Equal(t, personaltoken.Hash(value), hash)
		return db.UserToken{ID: "existing", UserID: 7, Name: personaltoken.ReservedOwnerTokenName}, nil
	}

	res, err := seedownertoken.Resolve(context.Background(), env, "owner@example.com", value)
	require.NoError(t, err)
	require.Equal(t, seedownertoken.OutcomeFound, res.Outcome)
	require.Empty(t, env.InsertUserTokenCalls())
}

// A revoked row is a deliberate kill switch. A restart must not resurrect it,
// or the only way to stop the fleet without downtime is worth nothing.
func TestLeavesARevokedRowRevoked(t *testing.T) {
	revoked := time.Now()
	env := baseEnv()
	env.UserTokenByHashAnyFunc = func(_ context.Context, _ string) (db.UserToken, error) {
		return db.UserToken{ID: "existing", UserID: 7, RevokedAt: &revoked}, nil
	}

	res, err := seedownertoken.Resolve(context.Background(), env, "owner@example.com", value)
	require.NoError(t, err)
	require.Equal(t, seedownertoken.OutcomeLeftRevoked, res.Outcome)
	require.Empty(t, env.InsertUserTokenCalls(), "a revoked row must not be re-inserted")
}

// Changing the value is the whole rotation story: the new row goes in and the
// same pass revokes whatever carried the reserved name before it.
func TestRevokesTheSupersededRows(t *testing.T) {
	env := baseEnv()
	env.RevokeSupersededOwnerTokensFunc = func(_ context.Context, arg db.RevokeSupersededOwnerTokensParams) (int64, error) {
		require.Equal(t, int64(7), arg.UserID)
		require.Equal(t, personaltoken.ReservedOwnerTokenName, arg.Name)
		require.Equal(t, personaltoken.Hash(value), arg.KeepHash)
		return 2, nil
	}

	res, err := seedownertoken.Resolve(context.Background(), env, "owner@example.com", value)
	require.NoError(t, err)
	require.Equal(t, int64(2), res.RevokedSuperseded)
}

// Clearing the value turns the feature off, which must revoke the live row
// rather than leave a credential nobody configured any more.
func TestEmptyValueRevokesEveryLiveSeededRow(t *testing.T) {
	env := baseEnv()
	env.RevokeSupersededOwnerTokensFunc = func(_ context.Context, arg db.RevokeSupersededOwnerTokensParams) (int64, error) {
		require.Empty(t, arg.KeepHash, "with no value configured nothing is kept")
		return 1, nil
	}

	res, err := seedownertoken.Resolve(context.Background(), env, "owner@example.com", "")
	require.NoError(t, err)
	require.Equal(t, seedownertoken.OutcomeDisabled, res.Outcome)
	require.Equal(t, int64(1), res.RevokedSuperseded)
	require.Empty(t, env.UserTokenByHashAnyCalls())
	require.Empty(t, env.InsertUserTokenCalls())
}

// No owner and no value is a stock instance: the seeder has nothing to attach a
// token to and must not fail the boot over it.
func TestNoOwnerAndNoValueIsASkip(t *testing.T) {
	env := baseEnv()

	res, err := seedownertoken.Resolve(context.Background(), env, "", "")
	require.NoError(t, err)
	require.Equal(t, seedownertoken.OutcomeSkipped, res.Outcome)
	require.Empty(t, env.UserByEmailCalls())
}

// A value with no owner to own it would be a silently dead credential.
func TestValueWithoutAnOwnerFails(t *testing.T) {
	env := baseEnv()

	_, err := seedownertoken.Resolve(context.Background(), env, "", value)
	require.Error(t, err)
	require.Contains(t, err.Error(), "owner")
}

func TestOwnerLookupFailureSurfaces(t *testing.T) {
	env := baseEnv()
	env.UserByEmailFunc = func(_ context.Context, _ string) (db.User, error) {
		return db.User{}, errors.New("db down")
	}

	_, err := seedownertoken.Resolve(context.Background(), env, "owner@example.com", value)
	require.Error(t, err)
	require.Contains(t, err.Error(), "db down")
}
