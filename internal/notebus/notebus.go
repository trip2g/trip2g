package notebus

import (
	"sync"
	"sync/atomic"

	"trip2g/internal/logger"
	"trip2g/internal/webhookutil"
)

// Change represents a single note change event.
type Change struct {
	PathID int64
	Path   string // used for remove events when note is no longer in NoteViews
	Event  string // "create", "update", "remove"
}

// Batch is a collection of changes published together.
type Batch struct {
	Changes []Change
}

// Stats holds bus metrics.
type Stats struct {
	Subscribers int
	Published   int64
	Dropped     int64
}

// Subscriber is a handle returned by Subscribe.
type Subscriber struct {
	Ch      <-chan Batch
	ch      chan Batch
	include []string
	exclude []string
}

// Bus is a pub/sub bus for note change events.
type Bus struct {
	log       logger.Logger
	mu        sync.RWMutex
	subs      map[*Subscriber]struct{}
	published atomic.Int64
	dropped   atomic.Int64
}

// New creates a new Bus.
func New(log logger.Logger) *Bus {
	return &Bus{
		log:  log,
		subs: make(map[*Subscriber]struct{}),
	}
}

// Subscribe registers a new subscriber with glob pattern filters.
func (b *Bus) Subscribe(includePatterns, excludePatterns []string, bufSize int) *Subscriber {
	ch := make(chan Batch, bufSize)
	sub := &Subscriber{
		Ch:      ch,
		ch:      ch,
		include: includePatterns,
		exclude: excludePatterns,
	}

	b.mu.Lock()
	b.subs[sub] = struct{}{}
	b.mu.Unlock()

	return sub
}

// Unsubscribe removes the subscriber so it stops receiving events.
//
// The channel is deliberately NOT closed: Publish snapshots subscribers under
// RLock and then sends after releasing it, so closing here would race that
// in-flight send and panic ("send on closed channel"), taking down the whole
// process. Consumers exit via ctx.Done() (see the subscription resolver), and
// the abandoned channel is reclaimed by GC once the subscriber is unreferenced.
func (b *Bus) Unsubscribe(sub *Subscriber) {
	b.mu.Lock()
	delete(b.subs, sub)
	b.mu.Unlock()
}

// Publish sends a filtered batch to all matching subscribers.
func (b *Bus) Publish(batch Batch) {
	b.published.Add(1)

	b.mu.RLock()
	subs := make([]*Subscriber, 0, len(b.subs))
	for sub := range b.subs {
		subs = append(subs, sub)
	}
	b.mu.RUnlock()

	for _, sub := range subs {
		filtered := filterBatch(batch, sub.include, sub.exclude)
		if len(filtered.Changes) == 0 {
			continue
		}
		select {
		case sub.ch <- filtered:
		default:
			b.dropped.Add(1)
			b.log.Warn("notebus: subscriber buffer full, dropping batch", "path", filtered.Changes[0].Path)
		}
	}
}

// Stats returns current bus metrics.
func (b *Bus) Stats() Stats {
	b.mu.RLock()
	count := len(b.subs)
	b.mu.RUnlock()

	return Stats{
		Subscribers: count,
		Published:   b.published.Load(),
		Dropped:     b.dropped.Load(),
	}
}

func filterBatch(batch Batch, include, exclude []string) Batch {
	var changes []Change
	for _, c := range batch.Changes {
		if len(include) > 0 && !webhookutil.MatchesAny(c.Path, include) {
			continue
		}
		if len(exclude) > 0 && webhookutil.MatchesAny(c.Path, exclude) {
			continue
		}
		changes = append(changes, c)
	}
	return Batch{Changes: changes}
}
