package notfoundtracker_test

import (
	"context"
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

	require.Len(t, env.calls.UpsertNotFoundHit, 0)

	require.Len(t, env.calls.UpsertNotFoundIPHit, 1)
	require.Equal(t, "127.0.0.1", env.calls.UpsertNotFoundIPHit[0].Arg.Ip)
	require.Equal(t, int64(300), env.calls.UpsertNotFoundIPHit[0].Arg.TotalHits)
}
