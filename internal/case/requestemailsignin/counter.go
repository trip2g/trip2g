package requestemailsignin

import (
	"sync"
	"time"
)

// SignInCounter tracks global sign-in code requests per hour window.
// Uses sync.Mutex to avoid TOCTOU race on window reset.
type SignInCounter struct {
	mu          sync.Mutex
	count       int64
	windowStart int64 // unix timestamp
}

// IncrementAndCheck increments the counter and returns true if the count
// exceeds the threshold. Resets the counter if the 1-hour window has elapsed.
func (c *SignInCounter) IncrementAndCheck(threshold int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Unix()
	if now-c.windowStart >= 3600 {
		c.count = 0
		c.windowStart = now
	}
	c.count++
	return c.count > threshold
}
