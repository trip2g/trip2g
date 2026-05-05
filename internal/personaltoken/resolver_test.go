package personaltoken_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"trip2g/internal/db"
	"trip2g/internal/personaltoken"
	"trip2g/internal/usertoken"
)

// --- minimal hand-rolled mock for Env ---

type mockEnv struct {
	mu sync.Mutex

	tokenByHashFunc         func(ctx context.Context, hash string) (db.UserToken, error)
	adminByUserIDFunc       func(ctx context.Context, userID int64) (db.Admin, error)
	updateLastUsedFunc      func(ctx context.Context, id string) error
	tokenByHashCalls        int
	updateLastUsedCalls     int
}

func (m *mockEnv) UserTokenByHash(ctx context.Context, hash string) (db.UserToken, error) {
	m.mu.Lock()
	m.tokenByHashCalls++
	m.mu.Unlock()
	return m.tokenByHashFunc(ctx, hash)
}

func (m *mockEnv) AdminByUserID(ctx context.Context, userID int64) (db.Admin, error) {
	return m.adminByUserIDFunc(ctx, userID)
}

func (m *mockEnv) UpdateUserTokenLastUsedAt(ctx context.Context, id string) error {
	m.mu.Lock()
	m.updateLastUsedCalls++
	m.mu.Unlock()
	if m.updateLastUsedFunc != nil {
		return m.updateLastUsedFunc(ctx, id)
	}
	return nil
}

func validToken(userID int64) db.UserToken {
	return db.UserToken{
		ID:     "tok-1",
		UserID: userID,
	}
}

func adminEnv(userID int64) *mockEnv {
	return &mockEnv{
		tokenByHashFunc: func(_ context.Context, _ string) (db.UserToken, error) {
			return validToken(userID), nil
		},
		adminByUserIDFunc: func(_ context.Context, _ int64) (db.Admin, error) {
			return db.Admin{}, nil
		},
	}
}

func userEnv(userID int64) *mockEnv {
	return &mockEnv{
		tokenByHashFunc: func(_ context.Context, _ string) (db.UserToken, error) {
			return validToken(userID), nil
		},
		adminByUserIDFunc: func(_ context.Context, _ int64) (db.Admin, error) {
			return db.Admin{}, errors.New("sql: no rows")
		},
	}
}

func notFoundEnv() *mockEnv {
	return &mockEnv{
		tokenByHashFunc: func(_ context.Context, _ string) (db.UserToken, error) {
			return db.UserToken{}, errors.New("sql: no rows")
		},
		adminByUserIDFunc: func(_ context.Context, _ int64) (db.Admin, error) {
			return db.Admin{}, nil
		},
	}
}

func TestResolver_ValidToken_AdminRole(t *testing.T) {
	env := adminEnv(42)
	r := personaltoken.NewResolver(env)
	data, err := r.Resolve(context.Background(), personaltoken.Generate())
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, 42, data.ID)
	require.Equal(t, "admin", data.Role)
}

func TestResolver_ValidToken_UserRole(t *testing.T) {
	env := userEnv(7)
	r := personaltoken.NewResolver(env)
	data, err := r.Resolve(context.Background(), personaltoken.Generate())
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, 7, data.ID)
	require.Equal(t, "user", data.Role)
}

func TestResolver_NotFound_ReturnsErrInvalidToken(t *testing.T) {
	env := notFoundEnv()
	r := personaltoken.NewResolver(env)
	_, err := r.Resolve(context.Background(), personaltoken.Generate())
	require.ErrorIs(t, err, personaltoken.ErrInvalidToken)
}

func TestResolver_CacheHit_SecondCallNoDBLookup(t *testing.T) {
	env := adminEnv(1)
	r := personaltoken.NewResolver(env)
	tok := personaltoken.Generate()

	_, err := r.Resolve(context.Background(), tok)
	require.NoError(t, err)

	_, err = r.Resolve(context.Background(), tok)
	require.NoError(t, err)

	// DB should only be called once
	env.mu.Lock()
	calls := env.tokenByHashCalls
	env.mu.Unlock()
	require.Equal(t, 1, calls, "second call within 30s should hit cache")
}

func TestResolver_ThrottleSkipsLastUsedWithin5Min(t *testing.T) {
	env := adminEnv(1)
	r := personaltoken.NewResolver(env)
	tok := personaltoken.Generate()

	_, err := r.Resolve(context.Background(), tok)
	require.NoError(t, err)
	// Small sleep to allow goroutine to run
	time.Sleep(20 * time.Millisecond)

	env.mu.Lock()
	firstCalls := env.updateLastUsedCalls
	env.mu.Unlock()
	require.Equal(t, 1, firstCalls)

	// Second call within 5 minutes — should NOT write again
	_, err = r.Resolve(context.Background(), tok)
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)

	env.mu.Lock()
	secondCalls := env.updateLastUsedCalls
	env.mu.Unlock()
	require.Equal(t, 1, secondCalls, "throttle: second call within 5min must not write last_used_at")
}

func TestResolver_IsPersonal_Filter(t *testing.T) {
	require.True(t, personaltoken.IsPersonal("t2g_abc"))
	require.False(t, personaltoken.IsPersonal("eyJhbGciOiJSUzI1NiJ9.e30.sig"))
}

// compile-time interface check.
var _ personaltoken.Env = (*mockEnv)(nil)

// satisfy usertoken import.
var _ *usertoken.Data = (*usertoken.Data)(nil)
