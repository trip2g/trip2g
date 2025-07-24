package handletgupdate

import (
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter.
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[int64]*bucket
	maxReqs int           // Maximum requests
	window  time.Duration // Time window
	cleanup time.Duration // Cleanup interval
}

type bucket struct {
	requests []time.Time
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter.
// maxReqs: maximum number of requests per window.
// window: time window duration.
func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[int64]*bucket),
		maxReqs: maxReqs,
		window:  window,
		cleanup: window * 2, // Cleanup old buckets after 2x window
	}

	// Start cleanup goroutine
	go rl.startCleanup()

	return rl
}

// Allow checks if a request from userID should be allowed.
func (rl *RateLimiter) Allow(userID int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get or create bucket for user
	b, exists := rl.buckets[userID]
	if !exists {
		b = &bucket{
			requests: make([]time.Time, 0),
			lastSeen: now,
		}
		rl.buckets[userID] = b
	}

	b.lastSeen = now

	// Remove old requests
	validRequests := make([]time.Time, 0, len(b.requests))
	for _, reqTime := range b.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	b.requests = validRequests

	// Check if under limit
	if len(b.requests) >= rl.maxReqs {
		return false
	}

	// Add current request
	b.requests = append(b.requests, now)
	return true
}

// startCleanup removes old, unused buckets.
func (rl *RateLimiter) startCleanup() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.cleanup)

		for userID, bucket := range rl.buckets {
			if bucket.lastSeen.Before(cutoff) {
				delete(rl.buckets, userID)
			}
		}
		rl.mu.Unlock()
	}
}
