package createusertoken_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/createusertoken"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createusertoken_test . Env

func TestResolve_Success(t *testing.T) {
	ctx := context.Background()

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 42, Role: "user"}, nil
		},
		CountActiveUserTokensByUserIDFunc: func(_ context.Context, userID int64) (int64, error) {
			require.Equal(t, int64(42), userID)
			return 3, nil
		},
		GenerateUniqIDFunc: func() string {
			return "test-uuid"
		},
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
	}

	result, err := createusertoken.Resolve(ctx, env, createusertoken.Input{
		Name:      "Claude Desktop",
		ExpiresIn: nil,
	})
	require.NoError(t, err)

	payload, ok := result.(*createusertoken.SuccessPayload)
	require.True(t, ok, "expected SuccessPayload, got %T", result)
	require.NotEmpty(t, payload.PlaintextToken)
	require.Greater(t, len(payload.PlaintextToken), 4)
	require.Equal(t, "test-uuid", payload.Token.ID)
	require.Equal(t, int64(42), payload.Token.UserID)
	require.Equal(t, "Claude Desktop", payload.Token.Name)
	require.Nil(t, payload.Token.ExpiresAt)
}

func TestResolve_WithExpiry(t *testing.T) {
	ctx := context.Background()

	var insertedExpiresAt *time.Time

	expiry := 90 * 24 * time.Hour
	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 1, Role: "user"}, nil
		},
		CountActiveUserTokensByUserIDFunc: func(_ context.Context, _ int64) (int64, error) {
			return 0, nil
		},
		GenerateUniqIDFunc: func() string { return "uuid-expiry" },
		InsertUserTokenFunc: func(_ context.Context, arg db.InsertUserTokenParams) (db.UserToken, error) {
			insertedExpiresAt = arg.ExpiresAt
			return db.UserToken{
				ID:        arg.ID,
				UserID:    arg.UserID,
				Name:      arg.Name,
				ExpiresAt: arg.ExpiresAt,
			}, nil
		},
	}

	before := time.Now()
	result, err := createusertoken.Resolve(ctx, env, createusertoken.Input{
		Name:      "Test",
		ExpiresIn: &expiry,
	})
	after := time.Now()

	require.NoError(t, err)
	_, ok := result.(*createusertoken.SuccessPayload)
	require.True(t, ok)
	require.NotNil(t, insertedExpiresAt)
	require.True(t, insertedExpiresAt.After(before.Add(expiry-time.Second)), "expires_at too early")
	require.True(t, insertedExpiresAt.Before(after.Add(expiry+time.Second)), "expires_at too late")
}

func TestResolve_LimitExceeded(t *testing.T) {
	ctx := context.Background()

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 7, Role: "user"}, nil
		},
		CountActiveUserTokensByUserIDFunc: func(_ context.Context, _ int64) (int64, error) {
			return 10, nil
		},
		GenerateUniqIDFunc: func() string { return "should-not-be-called" },
		InsertUserTokenFunc: func(_ context.Context, _ db.InsertUserTokenParams) (db.UserToken, error) {
			t.Fatal("InsertUserToken should not be called when limit is exceeded")
			return db.UserToken{}, nil
		},
	}

	result, err := createusertoken.Resolve(ctx, env, createusertoken.Input{Name: "Too Many"})
	require.NoError(t, err)

	errPayload, ok := result.(*createusertoken.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.NotEmpty(t, errPayload.Message)
}

func TestResolve_Unauthenticated(t *testing.T) {
	ctx := context.Background()

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) {
			return nil, nil
		},
		CountActiveUserTokensByUserIDFunc: func(_ context.Context, _ int64) (int64, error) {
			t.Fatal("should not be called")
			return 0, nil
		},
		GenerateUniqIDFunc: func() string { return "" },
		InsertUserTokenFunc: func(_ context.Context, _ db.InsertUserTokenParams) (db.UserToken, error) {
			t.Fatal("should not be called")
			return db.UserToken{}, nil
		},
	}

	result, err := createusertoken.Resolve(ctx, env, createusertoken.Input{Name: "Test"})
	require.NoError(t, err)
	errPayload, ok := result.(*createusertoken.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", result)
	require.NotEmpty(t, errPayload.Message)
}
