package createusertoken_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/createusertoken"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createusertoken_test . Env

func baseEnv(t *testing.T) *EnvMock {
	t.Helper()

	return &EnvMock{
		CountActiveUserTokensByUserIDFunc: func(_ context.Context, _ int64) (int64, error) {
			return 0, nil
		},
		GenerateUniqIDFunc: func() string { return "test-uuid" },
		InsertUserTokenFunc: func(_ context.Context, arg db.InsertUserTokenParams) (db.UserToken, error) {
			return db.UserToken{
				ID:          arg.ID,
				UserID:      arg.UserID,
				Name:        arg.Name,
				TokenHash:   arg.TokenHash,
				TokenPrefix: arg.TokenPrefix,
				Scope:       arg.Scope,
				ExpiresAt:   arg.ExpiresAt,
			}, nil
		},
		PublicURLFunc: func() string { return "https://example.com" },
	}
}

func TestResolveMintsForTheUserItWasGiven(t *testing.T) {
	ctx := context.Background()

	env := baseEnv(t)
	env.CountActiveUserTokensByUserIDFunc = func(_ context.Context, userID int64) (int64, error) {
		require.Equal(t, int64(42), userID)
		return 3, nil
	}

	result, err := createusertoken.Resolve(ctx, env, 42, createusertoken.Input{Name: "Claude Desktop"})
	require.NoError(t, err)

	payload, ok := result.(*model.CreateUserTokenPayload)
	require.True(t, ok, "expected CreateUserTokenPayload, got %T", result)
	require.NotEmpty(t, payload.PlaintextToken)
	require.Greater(t, len(payload.PlaintextToken), 4)
	require.Equal(t, "test-uuid", payload.Token.ID)
	require.Equal(t, int64(42), payload.Token.UserID)
	require.Equal(t, "Claude Desktop", payload.Token.Name)
	require.Nil(t, payload.Token.ExpiresAt)
}

func TestResolveReturnsInstructionsThatAlreadyCarryTheToken(t *testing.T) {
	ctx := context.Background()

	env := baseEnv(t)

	result, err := createusertoken.Resolve(ctx, env, 1, createusertoken.Input{Name: "Agent"})
	require.NoError(t, err)

	payload, ok := result.(*model.CreateUserTokenPayload)
	require.True(t, ok)
	require.Contains(t, payload.Instructions, "https://example.com/_system/mcp")
	require.Contains(t, payload.Instructions, "Bearer "+payload.PlaintextToken,
		"the text is meant to work as-is, so the token is already in it")
}

func TestResolveTurnsExpiryDaysIntoATimestamp(t *testing.T) {
	ctx := context.Background()

	var insertedExpiresAt *time.Time

	days := int32(90)
	expiry := 90 * 24 * time.Hour

	env := baseEnv(t)
	env.GenerateUniqIDFunc = func() string { return "uuid-expiry" }
	env.InsertUserTokenFunc = func(_ context.Context, arg db.InsertUserTokenParams) (db.UserToken, error) {
		insertedExpiresAt = arg.ExpiresAt
		return db.UserToken{ID: arg.ID, UserID: arg.UserID, Name: arg.Name, ExpiresAt: arg.ExpiresAt}, nil
	}

	before := time.Now()
	result, err := createusertoken.Resolve(ctx, env, 1, createusertoken.Input{
		Name:          "Test",
		ExpiresInDays: &days,
	})
	after := time.Now()

	require.NoError(t, err)
	_, ok := result.(*model.CreateUserTokenPayload)
	require.True(t, ok)
	require.NotNil(t, insertedExpiresAt)
	require.True(t, insertedExpiresAt.After(before.Add(expiry-time.Second)), "expires_at too early")
	require.True(t, insertedExpiresAt.Before(after.Add(expiry+time.Second)), "expires_at too late")
}

func TestResolveRefusesOnceTheUserHoldsTenActiveTokens(t *testing.T) {
	ctx := context.Background()

	env := baseEnv(t)
	env.CountActiveUserTokensByUserIDFunc = func(_ context.Context, _ int64) (int64, error) {
		return 10, nil
	}
	env.InsertUserTokenFunc = func(_ context.Context, _ db.InsertUserTokenParams) (db.UserToken, error) {
		t.Fatal("InsertUserToken must not be called when the limit is exceeded")
		return db.UserToken{}, nil
	}

	result, err := createusertoken.Resolve(ctx, env, 7, createusertoken.Input{Name: "Too Many"})
	require.NoError(t, err)

	errPayload, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.Equal(t, "token_limit_exceeded", errPayload.Message)
}
