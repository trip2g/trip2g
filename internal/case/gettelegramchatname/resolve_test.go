package gettelegramchatname_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"trip2g/internal/case/gettelegramchatname"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
)

type testEnv struct {
	now time.Time

	cacheRow        db.TelegramChatUsername
	cacheErr        error
	upsertArgs      []db.UpsertTelegramChatUsernameParams
	markRefreshArgs []db.MarkTelegramChatUsernameRefreshRequestedParams
	botResult       gettelegramchatname.LookupResult
	botFound        bool
	botErr          error
	accountResult   gettelegramchatname.LookupResult
	accountFound    bool
	accountErr      error
	botCalls        int
	accountCalls    int
}

func (e *testEnv) Logger() logger.Logger { return &logger.DummyLogger{} }
func (e *testEnv) Now() time.Time        { return e.now }

func (e *testEnv) GetTelegramChatUsernameByChatID(ctx context.Context, chatID int64) (db.TelegramChatUsername, error) {
	return e.cacheRow, e.cacheErr
}

func (e *testEnv) ListStaleTelegramChatUsernames(ctx context.Context, arg db.ListStaleTelegramChatUsernamesParams) ([]db.TelegramChatUsername, error) {
	return nil, nil
}

func (e *testEnv) UpsertTelegramChatUsername(ctx context.Context, arg db.UpsertTelegramChatUsernameParams) error {
	e.upsertArgs = append(e.upsertArgs, arg)
	return nil
}

func (e *testEnv) MarkTelegramChatUsernameRefreshRequested(ctx context.Context, arg db.MarkTelegramChatUsernameRefreshRequestedParams) error {
	e.markRefreshArgs = append(e.markRefreshArgs, arg)
	return nil
}

func (e *testEnv) MarkTelegramChatUsernameRefreshError(ctx context.Context, arg db.MarkTelegramChatUsernameRefreshErrorParams) error {
	return nil
}

func (e *testEnv) ResolveTelegramChatUsernameViaBot(ctx context.Context, chatID int64) (gettelegramchatname.LookupResult, bool, error) {
	e.botCalls++
	return e.botResult, e.botFound, e.botErr
}

func (e *testEnv) ResolveTelegramChatUsernameViaAccount(ctx context.Context, chatID int64) (gettelegramchatname.LookupResult, bool, error) {
	e.accountCalls++
	return e.accountResult, e.accountFound, e.accountErr
}

func TestResolve_ReturnsFreshCachedUsername(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	env := &testEnv{
		now: now,
		cacheRow: db.TelegramChatUsername{
			TelegramChatID: -1001234567890,
			Username:       "publicchannel",
			Title:          "Public Channel",
			RefreshedAt:    now.Add(-24 * time.Hour),
		},
	}

	username, err := gettelegramchatname.Resolve(context.Background(), env, -1001234567890)
	require.NoError(t, err)
	require.Equal(t, "publicchannel", username)
	require.Zero(t, env.botCalls)
	require.Zero(t, env.accountCalls)
	require.Empty(t, env.markRefreshArgs)
}

func TestResolve_ReturnsStaleUsernameAndSchedulesRefresh(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	env := &testEnv{
		now: now,
		cacheRow: db.TelegramChatUsername{
			TelegramChatID: -1001234567890,
			Username:       "publicchannel",
			Title:          "Public Channel",
			RefreshedAt:    now.Add(-8 * 24 * time.Hour),
		},
	}

	username, err := gettelegramchatname.Resolve(context.Background(), env, -1001234567890)
	require.NoError(t, err)
	require.Equal(t, "publicchannel", username)
	require.Len(t, env.markRefreshArgs, 1)
	require.Equal(t, int64(-1001234567890), env.markRefreshArgs[0].TelegramChatID)
	require.Zero(t, env.botCalls)
	require.Zero(t, env.accountCalls)
}

func TestResolve_CacheMissResolvesViaBotFirst(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	env := &testEnv{
		now:      now,
		cacheErr: sql.ErrNoRows,
		botFound: true,
		botResult: gettelegramchatname.LookupResult{
			Username: "publicchannel",
			Title:    "Public Channel",
		},
	}

	username, err := gettelegramchatname.Resolve(context.Background(), env, -1001234567890)
	require.NoError(t, err)
	require.Equal(t, "publicchannel", username)
	require.Equal(t, 1, env.botCalls)
	require.Zero(t, env.accountCalls)
	require.Len(t, env.upsertArgs, 1)
	require.Equal(t, "publicchannel", env.upsertArgs[0].Username)
	require.Equal(t, "Public Channel", env.upsertArgs[0].Title)
}

func TestResolve_CacheMissFallsBackToAccount(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	env := &testEnv{
		now:          now,
		cacheErr:     sql.ErrNoRows,
		accountFound: true,
		accountResult: gettelegramchatname.LookupResult{
			Username: "accountchannel",
			Title:    "Account Channel",
		},
	}

	username, err := gettelegramchatname.Resolve(context.Background(), env, -1001234567890)
	require.NoError(t, err)
	require.Equal(t, "accountchannel", username)
	require.Equal(t, 1, env.botCalls)
	require.Equal(t, 1, env.accountCalls)
	require.Len(t, env.upsertArgs, 1)
	require.Equal(t, "accountchannel", env.upsertArgs[0].Username)
}

func TestResolve_CacheMissStoresNegativeResult(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	env := &testEnv{
		now:      now,
		cacheErr: sql.ErrNoRows,
	}

	username, err := gettelegramchatname.Resolve(context.Background(), env, -1001234567890)
	require.NoError(t, err)
	require.Empty(t, username)
	require.Equal(t, 1, env.botCalls)
	require.Equal(t, 1, env.accountCalls)
	require.Len(t, env.upsertArgs, 1)
	require.Empty(t, env.upsertArgs[0].Username)
}
