package codellm

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

// idempotencyKeyHeader is the client-chosen key that names one logical chat
// call. Fleet's OpenAILLM sends the same key on every retry of a call, so a
// replay is served from the record instead of executing the blocks again — a
// network-level retry must never run a block with side effects twice.
const idempotencyKeyHeader = "Idempotency-Key"

// replayTTL is how long a recorded response stays servable. Fleet's retries
// follow the failure within seconds; the window only has to outlast them.
const replayTTL = 10 * time.Minute

// replayEntry is one keyed chat call: in flight until done is closed, then the
// recorded response.
type replayEntry struct {
	done   chan struct{}
	status int
	header http.Header
	body   []byte
	doneAt time.Time
}

// replayStore is the in-memory Idempotency-Key → response record, guarded by
// mu. Expired entries are swept on every claim, which bounds it to the keys
// seen within one TTL.
type replayStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	entries map[string]*replayEntry
}

func newReplayStore() *replayStore {
	return &replayStore{ttl: replayTTL, now: time.Now, entries: map[string]*replayEntry{}}
}

// claim returns the entry for key and whether the caller now owns its
// execution. A false owner must wait on entry.done, then serve the record.
func (s *replayStore) claim(key string) (*replayEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweep()
	if e, ok := s.entries[key]; ok {
		return e, false
	}
	e := &replayEntry{done: make(chan struct{})}
	s.entries[key] = e
	return e, true
}

func (s *replayStore) sweep() {
	cutoff := s.now().Add(-s.ttl)
	for key, e := range s.entries {
		select {
		case <-e.done:
			if e.doneAt.Before(cutoff) {
				delete(s.entries, key)
			}
		default:
		}
	}
}

// complete records the response the owner produced and releases the waiters.
func (s *replayStore) complete(e *replayEntry, rec *replayRecorder) {
	s.mu.Lock()
	e.status, e.header, e.body, e.doneAt = rec.status, rec.header, rec.body.Bytes(), s.now()
	s.mu.Unlock()
	close(e.done)
}

// serve writes the recorded response. Only valid once done is closed.
func (e *replayEntry) serve(w http.ResponseWriter) {
	for k, v := range e.header {
		w.Header()[k] = v
	}
	w.WriteHeader(e.status)
	_, _ = w.Write(e.body)
}

// replayRecorder captures the owner's response so it can be both sent and
// recorded.
type replayRecorder struct {
	status int
	header http.Header
	body   bytes.Buffer
}

func newReplayRecorder() *replayRecorder {
	return &replayRecorder{status: http.StatusOK, header: http.Header{}}
}

func (r *replayRecorder) Header() http.Header         { return r.header }
func (r *replayRecorder) WriteHeader(code int)        { r.status = code }
func (r *replayRecorder) Write(b []byte) (int, error) { return r.body.Write(b) }
