package main

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/model"
	"trip2g/internal/noteloader"

	"github.com/stretchr/testify/require"
)

type fakeBootLoader struct {
	takes time.Duration
	err   error
	nvs   *model.NoteViews
}

func (f *fakeBootLoader) Load(ctx context.Context, _ noteloader.LoadOptions) error {
	select {
	case <-time.After(f.takes):
	case <-ctx.Done():
		return ctx.Err()
	}
	if f.err != nil {
		return f.err
	}
	f.nvs = model.NewNoteViews()
	return nil
}

func (f *fakeBootLoader) NoteViews() *model.NoteViews {
	if f.nvs == nil {
		return model.NewNoteViews()
	}
	return f.nvs
}

func TestBootNoteLoaders_EachLoaderHasItsOwnBudget(t *testing.T) {
	budget := 200 * time.Millisecond
	live := &fakeBootLoader{takes: 150 * time.Millisecond}
	latest := &fakeBootLoader{takes: 150 * time.Millisecond}
	log := newSpyLogger()
	loaders := bootNoteLoaders{live: live, latest: latest, budget: budget, log: log}

	err := loaders.load(context.Background(), noteloader.LoadOptions{})
	require.NoError(t, err)
	require.NotNil(t, latest.nvs, "the latest loader must not inherit what the live loader spent")
	require.Equal(t, "info", log.levels["loaded latest notes"])
}

func TestBootNoteLoaders_LiveOverBudgetIsFatal(t *testing.T) {
	live := &fakeBootLoader{takes: time.Second}
	latest := &fakeBootLoader{}
	loaders := bootNoteLoaders{live: live, latest: latest, budget: 20 * time.Millisecond, log: newSpyLogger()}

	err := loaders.load(context.Background(), noteloader.LoadOptions{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, latest.nvs)
}
