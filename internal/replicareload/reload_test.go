package replicareload

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

func newTestReloader(env Env) *ReplicaReload {
	return New(env, &logger.DummyLogger{}, time.Second)
}

func TestReloadIfChanged_FirstCallAlwaysReloads(t *testing.T) {
	sig := db.NotesReloadSignalRow{VersionGen: 5, HiddenCount: 0}
	latestCalls := 0
	liveCalls := 0
	var gotPartial bool
	env := &EnvMock{
		NotesReloadSignalFunc: func(ctx context.Context) (db.NotesReloadSignalRow, error) {
			return sig, nil
		},
		PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*model.NoteViews, error) {
			latestCalls++
			gotPartial = partial
			return nil, nil
		},
		PrepareLiveNotesFunc: func(ctx context.Context) (*model.NoteViews, error) {
			liveCalls++
			return nil, nil
		},
	}
	r := newTestReloader(env)
	reloaded, err := r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)
	require.Equal(t, 1, latestCalls)
	require.Equal(t, 1, liveCalls)
	require.Equal(t, sig, r.last)
	require.True(t, r.haveLast)
	// replica reload must skip the search-index rebuild
	require.True(t, gotPartial)
}

func TestReloadIfChanged_UnchangedSignalSkips(t *testing.T) {
	sig := db.NotesReloadSignalRow{VersionGen: 5, HiddenCount: 0}
	latestCalls := 0
	liveCalls := 0
	env := &EnvMock{
		NotesReloadSignalFunc: func(ctx context.Context) (db.NotesReloadSignalRow, error) {
			return sig, nil
		},
		PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*model.NoteViews, error) {
			latestCalls++
			return nil, nil
		},
		PrepareLiveNotesFunc: func(ctx context.Context) (*model.NoteViews, error) {
			liveCalls++
			return nil, nil
		},
	}
	r := newTestReloader(env)

	// first call
	reloaded, err := r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)
	require.Equal(t, 1, latestCalls)

	// second call with same signal — must skip
	reloaded, err = r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.False(t, reloaded)
	require.Equal(t, 1, latestCalls) // no additional calls
	require.Equal(t, 1, liveCalls)
}

func TestReloadIfChanged_VersionChangeReloads(t *testing.T) {
	gen := int64(5)
	env := &EnvMock{
		NotesReloadSignalFunc: func(ctx context.Context) (db.NotesReloadSignalRow, error) {
			return db.NotesReloadSignalRow{VersionGen: gen, HiddenCount: 0}, nil
		},
		PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*model.NoteViews, error) {
			return nil, nil
		},
		PrepareLiveNotesFunc: func(ctx context.Context) (*model.NoteViews, error) {
			return nil, nil
		},
	}
	r := newTestReloader(env)

	// first call with version 5
	reloaded, err := r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)

	// version bumped to 6
	gen = 6
	reloaded, err = r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)
	require.Equal(t, int64(6), r.last.VersionGen)
}

func TestReloadIfChanged_HiddenChangeReloads(t *testing.T) {
	hidden := int64(0)
	env := &EnvMock{
		NotesReloadSignalFunc: func(ctx context.Context) (db.NotesReloadSignalRow, error) {
			return db.NotesReloadSignalRow{VersionGen: 6, HiddenCount: hidden}, nil
		},
		PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*model.NoteViews, error) {
			return nil, nil
		},
		PrepareLiveNotesFunc: func(ctx context.Context) (*model.NoteViews, error) {
			return nil, nil
		},
	}
	r := newTestReloader(env)

	// establish baseline
	reloaded, err := r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)

	// hide a note — hidden_count changes, version_gen unchanged
	hidden = 1
	reloaded, err = r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)
	require.Equal(t, int64(1), r.last.HiddenCount)
}

func TestReloadIfChanged_SignalErrorNoReload(t *testing.T) {
	callCount := 0
	sigErr := errors.New("db error")
	env := &EnvMock{
		NotesReloadSignalFunc: func(ctx context.Context) (db.NotesReloadSignalRow, error) {
			callCount++
			if callCount == 1 {
				return db.NotesReloadSignalRow{}, sigErr
			}
			return db.NotesReloadSignalRow{VersionGen: 1}, nil
		},
		PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*model.NoteViews, error) {
			return nil, nil
		},
		PrepareLiveNotesFunc: func(ctx context.Context) (*model.NoteViews, error) {
			return nil, nil
		},
	}
	r := newTestReloader(env)

	// first call: signal errors
	reloaded, err := r.reloadIfChanged(context.Background())
	require.ErrorIs(t, err, sigErr)
	require.False(t, reloaded)
	require.False(t, r.haveLast)

	// second call: signal succeeds — must reload (haveLast still false)
	reloaded, err = r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)
}

func TestReloadIfChanged_PrepareLatestNotesError(t *testing.T) {
	prepErr := errors.New("prepare error")
	calls := 0
	env := &EnvMock{
		NotesReloadSignalFunc: func(ctx context.Context) (db.NotesReloadSignalRow, error) {
			return db.NotesReloadSignalRow{VersionGen: 1}, nil
		},
		PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*model.NoteViews, error) {
			calls++
			if calls == 1 {
				return nil, prepErr
			}
			return nil, nil
		},
		PrepareLiveNotesFunc: func(ctx context.Context) (*model.NoteViews, error) {
			return nil, nil
		},
	}
	r := newTestReloader(env)

	// first call: PrepareLatestNotes errors
	reloaded, err := r.reloadIfChanged(context.Background())
	require.ErrorIs(t, err, prepErr)
	require.False(t, reloaded)
	require.False(t, r.haveLast)

	// second call: succeeds and retries (haveLast still false)
	reloaded, err = r.reloadIfChanged(context.Background())
	require.NoError(t, err)
	require.True(t, reloaded)
}
