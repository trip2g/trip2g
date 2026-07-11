// Package fedinstr caches federated knowledge-base instructions per kb_id path.
// A federated instructions lookup forwards through the federation hop chain, so
// repeat calls (and already-visited routes) are served from here instead of
// re-forwarding to the peer.
package fedinstr

import (
	"sync"
	"time"

	"trip2g/internal/model"
)

const (
	DefaultMaxEntries = 256
	DefaultTTL        = 10 * time.Minute
)

type entry struct {
	result   model.FederationResult
	storedAt time.Time
}

// Cache is a bounded, TTL'd, concurrency-safe store of federated instructions
// keyed by the full kb_id path (e.g. "philosophers/nietzsche").
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]entry
	ttl        time.Duration
	maxEntries int
	now        func() time.Time
}

func New() *Cache {
	return &Cache{
		entries:    make(map[string]entry),
		ttl:        DefaultTTL,
		maxEntries: DefaultMaxEntries,
		now:        time.Now,
	}
}

// CachedFederatedInstructions returns a fresh cached result for kbID, or ok=false
// when absent or stale.
func (c *Cache) CachedFederatedInstructions(kbID string) (model.FederationResult, bool) {
	c.mu.RLock()
	e, ok := c.entries[kbID]
	c.mu.RUnlock()
	if !ok || c.now().Sub(e.storedAt) >= c.ttl {
		return model.FederationResult{}, false
	}
	return e.result, true
}

// StoreFederatedInstructions caches result under kbID, evicting the oldest entry
// when the cache is full.
func (c *Cache) StoreFederatedInstructions(kbID string, result model.FederationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[kbID]; !exists && len(c.entries) >= c.maxEntries {
		c.evictOldestLocked()
	}
	c.entries[kbID] = entry{result: result, storedAt: c.now()}
}

func (c *Cache) evictOldestLocked() {
	var oldestKey string
	var oldestAt time.Time
	for k, e := range c.entries {
		if oldestKey == "" || e.storedAt.Before(oldestAt) {
			oldestKey = k
			oldestAt = e.storedAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}
