package notfoundtracker

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger
	ListAllNotFoundIgnoredPatterns(ctx context.Context) ([]db.NotFoundIgnoredPattern, error)
	ListActiveNotFoundIPHits(ctx context.Context) ([]db.NotFoundIpHit, error)
	UpsertNotFoundHit(ctx context.Context, path string) error
	UpsertNotFoundIPHit(ctx context.Context, arg db.UpsertNotFoundIPHitParams) error
}

type Tracker struct {
	mu     sync.RWMutex
	env    Env
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker

	ignorePatters []*regexp.Regexp
	ipHits        map[string]int64
}

const maxIPHits = 50

func New(ctx context.Context, env Env) (*Tracker, error) {
	ctxWithCancel, cancel := context.WithCancel(ctx)

	t := &Tracker{
		env:    env,
		ctx:    ctxWithCancel,
		cancel: cancel,
		ticker: time.NewTicker(time.Minute),
	}

	err := t.load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tracker: %w", err)
	}

	// Start background goroutine to dump state every minute
	go t.runDumpTicker()

	return t, nil
}

func (t *Tracker) Stop() {
	if t.ticker != nil {
		t.ticker.Stop()
	}

	if t.cancel != nil {
		t.cancel()
	}
}

func (t *Tracker) Dump() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	ctx := context.Background()

	// Sync all IP hits to database and reset memory counters
	for ip, hits := range t.ipHits {
		if hits > 0 {
			err := t.env.UpsertNotFoundIPHit(ctx, db.UpsertNotFoundIPHitParams{
				Ip:        ip,
				TotalHits: hits,
			})
			if err != nil {
				t.env.Logger().Error("failed to upsert IP hits", "ip", ip, "hits", hits, "error", err)
				continue
			}
		}
	}

	// Reset all IP hit counters in memory after syncing to DB
	// This allows IPs to start fresh each minute (unless they're already over limit in DB)
	t.ipHits = make(map[string]int64)

	return nil
}

func (t *Tracker) load(ctx context.Context) error {
	ignorePatterRows, err := t.env.ListAllNotFoundIgnoredPatterns(ctx)
	if err != nil {
		return fmt.Errorf("failed to list ignored patterns: %w", err)
	}

	ipHitRows, err := t.env.ListActiveNotFoundIPHits(ctx)
	if err != nil {
		return fmt.Errorf("failed to list IP hits: %w", err)
	}

	ignorePatters := make([]*regexp.Regexp, 0, len(ignorePatterRows))

	for _, row := range ignorePatterRows {
		pattern, err := regexp.Compile(row.Pattern)
		if err != nil {
			t.env.Logger().Warn("failed to compile pattern", "pattern", row.Pattern, "error", err)
			continue
		}

		ignorePatters = append(ignorePatters, pattern)
	}

	ipHits := make(map[string]int64, len(ipHitRows))

	for _, row := range ipHitRows {
		ipHits[row.Ip] = row.TotalHits
	}

	t.mu.Lock()
	t.ignorePatters = ignorePatters
	t.ipHits = ipHits
	t.mu.Unlock()

	return nil
}

func (t *Tracker) Track(path string, ip string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Increment IP hit counter first (for rate limiting)
	t.ipHits[ip]++

	// If IP has reached or exceeded max hits, don't process the path (rate limiting)
	if t.ipHits[ip] >= maxIPHits {
		return nil
	}

	// Check if path should be ignored
	for _, pattern := range t.ignorePatters {
		if pattern.MatchString(path) {
			return nil
		}
	}

	// Only insert path hit into database if IP hasn't reached limit
	ctx := context.Background()
	err := t.env.UpsertNotFoundHit(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to insert not found hit: %w", err)
	}

	return nil
}

func (t *Tracker) runDumpTicker() {
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-t.ticker.C:
			err := t.Dump()
			if err != nil {
				t.env.Logger().Error("failed to dump tracker state", "error", err)
			}
		}
	}
}
