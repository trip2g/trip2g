package notfoundtracker_test

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/notfoundtracker"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg notfoundtracker_test . Env

func TestTracker_StopUpsertNotFoundHitsAfterLimit(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	for range 200 {
		tracker.Track("/test", "127.0.0.1")
	}

	err = tracker.Dump()
	require.NoError(t, err)

	require.Len(t, env.calls.UpsertNotFoundHit, 49)
	require.Equal(t, "/test", env.calls.UpsertNotFoundHit[0].Path)

	require.Len(t, env.calls.UpsertNotFoundIPHit, 1)
	require.Equal(t, "127.0.0.1", env.calls.UpsertNotFoundIPHit[0].Arg.Ip)
	require.Equal(t, int64(200), env.calls.UpsertNotFoundIPHit[0].Arg.TotalHits)
}

func TestTracker_StopUpsertNotFoundHitsAfterLimitWithExistsHistory(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{{Ip: "127.0.0.1", TotalHits: 100}}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	for range 200 {
		tracker.Track("/test", "127.0.0.1")
	}

	err = tracker.Dump()
	require.NoError(t, err)

	require.Empty(t, env.calls.UpsertNotFoundHit)

	require.Len(t, env.calls.UpsertNotFoundIPHit, 1)
	require.Equal(t, "127.0.0.1", env.calls.UpsertNotFoundIPHit[0].Arg.Ip)
	require.Equal(t, int64(300), env.calls.UpsertNotFoundIPHit[0].Arg.TotalHits)
}

func TestTracker_IgnorePatternsWork(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{
				{Pattern: `^/static/.*`},
				{Pattern: `.*\.png$`},
				{Pattern: `.*\.js$`},
			}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	// These should be ignored
	err = tracker.Track("/static/image.png", "127.0.0.1")
	require.NoError(t, err)
	err = tracker.Track("/favicon.png", "127.0.0.1")
	require.NoError(t, err)
	err = tracker.Track("/assets/app.js", "127.0.0.1")
	require.NoError(t, err)

	// This should not be ignored
	err = tracker.Track("/nonexistent-page", "127.0.0.1")
	require.NoError(t, err)

	err = tracker.Dump()
	require.NoError(t, err)

	// Only the non-ignored path should be tracked
	require.Len(t, env.calls.UpsertNotFoundHit, 1)
	require.Equal(t, "/nonexistent-page", env.calls.UpsertNotFoundHit[0].Path)

	// IP hits should still be tracked regardless of ignored patterns
	require.Len(t, env.calls.UpsertNotFoundIPHit, 1)
	require.Equal(t, "127.0.0.1", env.calls.UpsertNotFoundIPHit[0].Arg.Ip)
	require.Equal(t, int64(4), env.calls.UpsertNotFoundIPHit[0].Arg.TotalHits)
}

func TestTracker_InvalidRegexPattern(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{
				{Pattern: `[invalid-regex`}, // Invalid regex
				{Pattern: `^/static/.*`},    // Valid regex
			}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err) // Should not error, just skip invalid patterns

	// Should be ignored by valid pattern
	err = tracker.Track("/static/image.png", "127.0.0.1")
	require.NoError(t, err)

	// Should not be ignored (invalid pattern doesn't work)
	err = tracker.Track("/some-path", "127.0.0.1")
	require.NoError(t, err)

	err = tracker.Dump()
	require.NoError(t, err)

	// Only the non-static path should be tracked
	require.Len(t, env.calls.UpsertNotFoundHit, 1)
	require.Equal(t, "/some-path", env.calls.UpsertNotFoundHit[0].Path)
}

func TestTracker_MultipleIPs(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	// Different IPs, some hitting limits, some not
	for range 60 {
		err = tracker.Track("/test1", "192.168.1.1") // Over limit
		require.NoError(t, err)
	}
	for range 30 {
		err = tracker.Track("/test2", "192.168.1.2") // Under limit
		require.NoError(t, err)
	}
	for range 100 {
		err = tracker.Track("/test3", "192.168.1.3") // Way over limit
		require.NoError(t, err)
	}

	err = tracker.Dump()
	require.NoError(t, err)

	// Should track hits only up to the limit per IP
	// IP1: 49 hits, IP2: 30 hits, IP3: 49 hits = 128 total path hits
	require.Len(t, env.calls.UpsertNotFoundHit, 128)

	// Should have 3 IP hit records
	require.Len(t, env.calls.UpsertNotFoundIPHit, 3)

	// Verify IP hit totals
	ipHits := make(map[string]int64)
	for _, call := range env.calls.UpsertNotFoundIPHit {
		ipHits[call.Arg.Ip] = call.Arg.TotalHits
	}
	require.Equal(t, int64(60), ipHits["192.168.1.1"])
	require.Equal(t, int64(30), ipHits["192.168.1.2"])
	require.Equal(t, int64(100), ipHits["192.168.1.3"])
}

func TestTracker_DumpResetsMemory(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	// Make some requests
	for range 10 {
		err = tracker.Track("/test", "127.0.0.1")
		require.NoError(t, err)
	}

	err = tracker.Dump()
	require.NoError(t, err)

	// Clear the mock calls to test second round
	env.calls.UpsertNotFoundHit = nil
	env.calls.UpsertNotFoundIPHit = nil

	// Make more requests - should start fresh since memory was reset
	for range 15 {
		err = tracker.Track("/test", "127.0.0.1")
		require.NoError(t, err)
	}

	err = tracker.Dump()
	require.NoError(t, err)

	// Should see 15 new path hits and IP total of 15
	require.Len(t, env.calls.UpsertNotFoundHit, 15)
	require.Len(t, env.calls.UpsertNotFoundIPHit, 1)
	require.Equal(t, int64(15), env.calls.UpsertNotFoundIPHit[0].Arg.TotalHits)
}

func TestTracker_LoadInitializationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("error loading ignored patterns", func(t *testing.T) {
		env := &EnvMock{
			LoggerFunc: func() logger.Logger {
				return &logger.TestLogger{}
			},
			ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
				return nil, errors.New("test error")
			},
		}

		_, err := notfoundtracker.New(ctx, env)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to list ignored patterns")
	})

	t.Run("error loading IP hits", func(t *testing.T) {
		env := &EnvMock{
			LoggerFunc: func() logger.Logger {
				return &logger.TestLogger{}
			},
			ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
				return []db.NotFoundIgnoredPattern{}, nil
			},
			ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
				return nil, errors.New("test error")
			},
		}

		_, err := notfoundtracker.New(ctx, env)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to list IP hits")
	})
}

func TestTracker_DatabaseErrors(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return errors.New("test error") // Simulate DB error
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	// Should return error when DB operation fails
	err = tracker.Track("/test", "127.0.0.1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to insert not found hit")
}

func TestTracker_StopCleanup(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	// Should not panic
	tracker.Stop()
	tracker.Stop() // Should be safe to call multiple times
}

func TestTracker_EmptyDump(t *testing.T) {
	ctx := context.Background()
	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		ListAllNotFoundIgnoredPatternsFunc: func(ctx context.Context) ([]db.NotFoundIgnoredPattern, error) {
			return []db.NotFoundIgnoredPattern{}, nil
		},
		ListActiveNotFoundIPHitsFunc: func(ctx context.Context) ([]db.NotFoundIpHit, error) {
			return []db.NotFoundIpHit{}, nil
		},
		UpsertNotFoundHitFunc: func(ctx context.Context, path string) error {
			return nil
		},
		UpsertNotFoundIPHitFunc: func(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error {
			return nil
		},
	}

	tracker, err := notfoundtracker.New(ctx, env)
	require.NoError(t, err)

	// Dump without any tracking should not call any upsert methods
	err = tracker.Dump()
	require.NoError(t, err)

	require.Empty(t, env.calls.UpsertNotFoundHit)
	require.Empty(t, env.calls.UpsertNotFoundIPHit)
}
