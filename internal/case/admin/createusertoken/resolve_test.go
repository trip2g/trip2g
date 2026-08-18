package createusertoken_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	createusertoken "trip2g/internal/case/admin/createusertoken"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createusertoken_test . Env

func baseEnv(t *testing.T) *EnvMock {
	t.Helper()

	return &EnvMock{
		CurrentAdminUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 1, Role: "admin"}, nil
		},
		UserByIDFunc: func(_ context.Context, id int64) (db.User, error) {
			return db.User{ID: id}, nil
		},
		AuditLoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
		CountActiveUserTokensByUserIDFunc: func(_ context.Context, _ int64) (int64, error) {
			return 0, nil
		},
		GenerateUniqIDFunc: func() string { return "admin-uuid" },
		InsertUserTokenFunc: func(_ context.Context, arg db.InsertUserTokenParams) (db.UserToken, error) {
			return db.UserToken{ID: arg.ID, UserID: arg.UserID, Name: arg.Name, ExpiresAt: arg.ExpiresAt}, nil
		},
		PublicURLFunc: func() string { return "https://example.com" },
	}
}

func TestResolveMintsForTheTargetUserNotTheAdmin(t *testing.T) {
	ctx := context.Background()

	env := baseEnv(t)
	env.CountActiveUserTokensByUserIDFunc = func(_ context.Context, userID int64) (int64, error) {
		require.Equal(t, int64(77), userID, "the token belongs to the target, not to the acting admin")
		return 0, nil
	}

	result, err := createusertoken.Resolve(ctx, env, createusertoken.Input{UserID: 77, Name: "Colleague"})
	require.NoError(t, err)

	payload, ok := result.(*model.CreateUserTokenPayload)
	require.True(t, ok, "expected CreateUserTokenPayload, got %T", result)
	require.Equal(t, int64(77), payload.Token.UserID)
	require.NotEmpty(t, payload.PlaintextToken)
	require.Contains(t, payload.Instructions, "/_system/mcp")
}

func TestResolveRefusesAnUnknownUserWithoutMintingAnything(t *testing.T) {
	ctx := context.Background()

	env := baseEnv(t)
	env.UserByIDFunc = func(_ context.Context, _ int64) (db.User, error) {
		return db.User{}, sql.ErrNoRows
	}
	env.InsertUserTokenFunc = func(_ context.Context, _ db.InsertUserTokenParams) (db.UserToken, error) {
		t.Fatal("a token must not be minted for a user that does not exist")
		return db.UserToken{}, nil
	}

	result, err := createusertoken.Resolve(ctx, env, createusertoken.Input{UserID: 999, Name: "Ghost"})
	require.NoError(t, err)

	errPayload, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.NotEmpty(t, errPayload.ByFields)
}

func TestResolveRefusesAnExpiryBeyondTheCap(t *testing.T) {
	ctx := context.Background()

	days := int32(400)

	env := baseEnv(t)
	env.InsertUserTokenFunc = func(_ context.Context, _ db.InsertUserTokenParams) (db.UserToken, error) {
		t.Fatal("a token must not be minted when the expiry is refused")
		return db.UserToken{}, nil
	}

	result, err := createusertoken.Resolve(ctx, env, createusertoken.Input{
		UserID:        5,
		Name:          "Too Long",
		ExpiresInDays: &days,
	})
	require.NoError(t, err)

	errPayload, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.NotEmpty(t, errPayload.ByFields)
}
