package db

import (
	"runtime"
	"sort"
	"sync"
	"time"
)

// WriteHolder is a dev-only diagnostic that records who currently occupies the
// single write connection: open write transactions and in-flight non-tx write
// statements. When the write pool is exhausted (in_use=1 max=1), its Snapshot
// names the holder with a goroutine stack and hold duration instead of leaving
// only "someone is blocking" warnings in the log.
//
// Capturing a stack per acquire has a cost, so the tracker is enabled only in
// dev mode; disabled (or nil) it is a no-op.
type WriteHolder struct {
	enabled bool

	mu      sync.Mutex
	nextID  uint64
	entries map[uint64]*writeHolderEntry
}

type writeHolderEntry struct {
	label string
	stack []byte
	since time.Time
}

// WriteHolderInfo is one snapshot entry.
type WriteHolderInfo struct {
	Label   string        `json:"label"`
	HeldFor time.Duration `json:"held_for"`
	Stack   string        `json:"stack"`
}

// NewWriteHolder creates a tracker. When enabled is false every method is a
// no-op (production path).
func NewWriteHolder(enabled bool) *WriteHolder {
	return &WriteHolder{
		enabled: enabled,
		entries: make(map[uint64]*writeHolderEntry),
	}
}

// Acquire records the calling goroutine as occupying the write connection and
// returns a release func. The release func is idempotent.
func (h *WriteHolder) Acquire(label string) func() {
	if h == nil || !h.enabled {
		return func() {}
	}

	buf := make([]byte, 16*1024)
	buf = buf[:runtime.Stack(buf, false)]

	h.mu.Lock()
	h.nextID++
	id := h.nextID
	h.entries[id] = &writeHolderEntry{
		label: label,
		stack: buf,
		since: time.Now(),
	}
	h.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.entries, id)
			h.mu.Unlock()
		})
	}
}

// Snapshot returns the current entries, oldest first. With a single write
// connection the oldest entry is the actual holder; younger ones are waiters
// queued behind it.
func (h *WriteHolder) Snapshot() []WriteHolderInfo {
	if h == nil || !h.enabled {
		return nil
	}

	h.mu.Lock()
	entries := make([]*writeHolderEntry, 0, len(h.entries))
	for _, e := range h.entries {
		entries = append(entries, e)
	}
	h.mu.Unlock()

	sort.Slice(entries, func(i, j int) bool { return entries[i].since.Before(entries[j].since) })

	now := time.Now()
	infos := make([]WriteHolderInfo, 0, len(entries))
	for _, e := range entries {
		infos = append(infos, WriteHolderInfo{
			Label:   e.label,
			HeldFor: now.Sub(e.since),
			Stack:   string(e.stack),
		})
	}

	return infos
}
