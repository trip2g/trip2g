package gitapi

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

// lockEnv wraps fakeEnv with a real serializing lock plus an overlap counter,
// so the test can detect whether two critical sections ever run concurrently.
type lockEnv struct {
	fakeEnv
	mu     sync.Mutex
	active int32
	maxObs int32
}

func (l *lockEnv) LockNoteWrites() {
	l.mu.Lock()
	n := atomic.AddInt32(&l.active, 1)
	if n > atomic.LoadInt32(&l.maxObs) {
		atomic.StoreInt32(&l.maxObs, n)
	}
}

func (l *lockEnv) UnlockNoteWrites() {
	atomic.AddInt32(&l.active, -1)
	l.mu.Unlock()
}

func TestMaterializeAndApplySerialize(t *testing.T) {
	env := &lockEnv{}
	env.fakeEnv.notes = []NoteSource{{Path: "a.md", Content: []byte("a")}}
	api := newTestAPI(t, env)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env.LockNoteWrites()
			_ = api.materialize(context.Background())
			env.UnlockNoteWrites()
		}()
	}
	wg.Wait()

	if env.maxObs > 1 {
		t.Fatalf("observed %d concurrent critical sections, want 1", env.maxObs)
	}
}
