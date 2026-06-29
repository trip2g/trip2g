// Package replicareload keeps a read replica's in-memory note cache fresh.
// A replica never runs the leader's write/reload path, so without this it would
// serve a boot-time snapshot forever. It polls a cheap note-specific signal and
// reloads the cache only when notes actually change.
package replicareload

import (
	"context"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

// Env is the minimal set of app capabilities this loop needs.
type Env interface {
	NotesReloadSignal(ctx context.Context) (db.NotesReloadSignalRow, error)
	PrepareLatestNotes(ctx context.Context, partial bool) (*model.NoteViews, error)
	PrepareLiveNotes(ctx context.Context) (*model.NoteViews, error)
}

// ReplicaReload polls a note-change signal and reloads the cache on change.
type ReplicaReload struct {
	env      Env
	log      logger.Logger
	interval time.Duration
	last     db.NotesReloadSignalRow
	haveLast bool
}

// New creates a new ReplicaReload.
func New(env Env, log logger.Logger, interval time.Duration) *ReplicaReload {
	return &ReplicaReload{env: env, log: log, interval: interval}
}

// Run polls until ctx is cancelled. Intended to be launched in its own goroutine.
func (r *ReplicaReload) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.reloadIfChanged(ctx); err != nil {
				r.log.Warn("replica note reload failed", "error", err)
			}
		}
	}
}

// reloadIfChanged reads the signal and, if it differs from the last seen value
// (or none seen yet), reloads the latest+live note caches. Returns whether a
// reload happened. Extracted for unit testing.
func (r *ReplicaReload) reloadIfChanged(ctx context.Context) (bool, error) {
	sig, err := r.env.NotesReloadSignal(ctx)
	if err != nil {
		return false, err
	}
	if r.haveLast && sig == r.last {
		return false, nil
	}
	// partial=true: skip the bleve search-index rebuild. A read replica serves the
	// public read path only (search queries go through GraphQL, which the replica
	// forwards to the leader), so rebuilding the index on every change is wasted
	// work and the index-write IO churn can wedge the instance under heavy ingestion.
	if _, err = r.env.PrepareLatestNotes(ctx, true); err != nil {
		return false, err
	}
	if _, err = r.env.PrepareLiveNotes(ctx); err != nil {
		return false, err
	}
	r.last = sig
	r.haveLast = true
	r.log.Info("replica note cache refreshed", "version_gen", sig.VersionGen, "hidden_count", sig.HiddenCount)
	return true, nil
}
