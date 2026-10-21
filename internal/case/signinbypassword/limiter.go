package signinbypassword

import (
	"sync"
	"time"
)

// AttemptLimiter counts attempts per key in a fixed window shared by all keys.
// The whole map is dropped when the window rolls over, so it never outgrows
// the keys seen within one window.
type AttemptLimiter struct {
	mu          sync.Mutex
	max         int
	window      time.Duration
	windowStart time.Time
	counts      map[string]int
}

func NewAttemptLimiter(maxAttempts int, window time.Duration) *AttemptLimiter {
	return &AttemptLimiter{
		max:    maxAttempts,
		window: window,
		counts: map[string]int{},
	}
}

// Allow records an attempt for key and reports whether it is within the limit.
func (l *AttemptLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.windowStart) >= l.window {
		l.windowStart = now
		l.counts = map[string]int{}
	}

	l.counts[key]++

	return l.counts[key] <= l.max
}
