// Package movebus is an in-process pub/sub bus for realtime reader movement
// events (note -> note navigation). Events are ephemeral: never persisted,
// delivered only to currently connected subscribers (the admin graph).
package movebus

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"trip2g/internal/logger"
)

// Move is a single reader navigation event.
type Move struct {
	FromPathID *int64 // nil = entry from outside the KB (direct link, search, external referrer)
	ToPathID   int64
	At         time.Time
	// SessionKey is an anonymous walk-continuity token: an hourly-rotated
	// HMAC over the viewer id (see Bus.SessionKey). Never a raw user id.
	SessionKey string
}

// Stats holds bus metrics.
type Stats struct {
	Subscribers int
	Published   int64
	Dropped     int64
}

// Subscriber is a handle returned by Subscribe.
type Subscriber struct {
	Ch   <-chan Move
	ch   chan Move
	done chan struct{} // closed by Unsubscribe; signals Publish to stop sending
}

// Bus is a pub/sub bus for reader movement events.
type Bus struct {
	log       logger.Logger
	mu        sync.RWMutex
	subs      map[*Subscriber]struct{}
	published atomic.Int64
	dropped   atomic.Int64

	keyMu   sync.Mutex
	key     []byte
	keyHour int64
}

// New creates a new Bus.
func New(log logger.Logger) *Bus {
	return &Bus{
		log:  log,
		subs: make(map[*Subscriber]struct{}),
	}
}

// Subscribe registers a new subscriber.
func (b *Bus) Subscribe(bufSize int) *Subscriber {
	ch := make(chan Move, bufSize)
	sub := &Subscriber{
		Ch:   ch,
		ch:   ch,
		done: make(chan struct{}),
	}

	b.mu.Lock()
	b.subs[sub] = struct{}{}
	b.mu.Unlock()

	return sub
}

// Unsubscribe removes the subscriber and signals Publish to stop sending to it.
//
// We close the subscriber's `done` channel rather than its data channel: Publish
// can be mid-send after snapshotting subscribers, and closing the data channel
// would race that send and panic ("send on closed channel"). Closing `done` is
// safe — only Unsubscribe closes it, exactly once (guarded by map membership) —
// and Publish selects on it to skip a subscriber that left after the snapshot.
func (b *Bus) Unsubscribe(sub *Subscriber) {
	b.mu.Lock()
	_, ok := b.subs[sub]
	delete(b.subs, sub)
	b.mu.Unlock()

	if ok {
		close(sub.done)
	}
}

// HasSubscribers reports whether anyone is currently listening. The render
// path checks it before doing any per-move work, so an idle instance pays
// nothing for this feature.
func (b *Bus) HasSubscribers() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs) > 0
}

// Publish sends the move to all subscribers, non-blocking: a slow dashboard
// can never back-pressure page renders — its events are dropped instead.
func (b *Bus) Publish(move Move) {
	b.published.Add(1)

	b.mu.RLock()
	subs := make([]*Subscriber, 0, len(b.subs))
	for sub := range b.subs {
		subs = append(subs, sub)
	}
	b.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub.ch <- move:
		case <-sub.done: // unsubscribed after the snapshot — skip
		default:
			b.dropped.Add(1)
			b.log.Warn("movebus: subscriber buffer full, dropping move", "toPathId", move.ToPathID)
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

// SessionKey derives the anonymous walk-continuity token for a viewer id:
// HMAC-SHA256 over the id with a random process key rotated hourly, truncated
// to 8 bytes. Within one hour the same viewer maps to the same token (one
// color per walking reader on the dashboard); across hours or restarts the
// mapping changes, so no stable identity ever leaves the process.
func (b *Bus) SessionKey(viewerID string, now time.Time) string {
	hour := now.Unix() / 3600

	b.keyMu.Lock()
	if b.key == nil || b.keyHour != hour {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			// crypto/rand failure is effectively fatal elsewhere; degrade to
			// a fresh zero key rotation rather than crashing the render path.
			b.log.Error("movebus: failed to rotate session key", "error", err)
		}
		b.key = key
		b.keyHour = hour
	}
	key := b.key
	b.keyMu.Unlock()

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(viewerID))
	return hex.EncodeToString(mac.Sum(nil)[:8])
}
