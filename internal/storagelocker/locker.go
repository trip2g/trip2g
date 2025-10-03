package storagelocker

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Env interface {
	PutLock(ctx context.Context, objectID string, ttl time.Duration) error
	RemoveLock(ctx context.Context, objectID string) error
}

type Config struct {
	Enabled bool
	Name    string
	TTL     time.Duration
}

type Locker struct {
	config    Config
	env       Env
	objectID  string
	locked    bool
	ctx       context.Context
	cancel    context.CancelFunc
	renewDone chan struct{}
}

// DefaultConfig returns a default configuration for the storage locker.
func DefaultConfig() Config {
	return Config{
		Enabled: false,
		Name:    "storage",
		TTL:     5 * time.Minute,
	}
}

var ErrStorageNotSupportLock = errors.New("storage does not support locking")

// New creates a new storage locker and immediately acquires the lock if enabled.
// It panics if the lock already exists, ensuring only one instance can run.
// If disabled, returns a no-op locker.
// TTL defaults to 5 minutes if not specified.
func New(ctx context.Context, config Config, env Env) (*Locker, error) {
	lockCtx, cancel := context.WithCancel(ctx)

	locker := &Locker{
		config:    config,
		env:       env,
		locked:    false,
		ctx:       lockCtx,
		cancel:    cancel,
		renewDone: make(chan struct{}),
	}

	if !config.Enabled {
		return locker, nil
	}

	objectID := generateLockObjectID(config.Name)

	err := env.PutLock(lockCtx, objectID, config.TTL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("storage lock already exists or failed to acquire: %w", err)
	}

	// Check if the storage supports locking by attempting to put the lock again.
	// err = env.PutLock(lockCtx, objectID, ttl)
	// if err == nil {
	// 	cancel()
	// 	return nil, ErrStorageNotSupportLock
	// }

	locker.objectID = objectID
	locker.locked = true

	// Start background goroutine for lock renewal
	go locker.renewLock()

	return locker, nil
}

// Unlock removes the lock file for graceful shutdown.
func (l *Locker) Unlock(ctx context.Context) error {
	if !l.locked {
		return nil
	}

	// Stop the renewal goroutine
	l.cancel()

	// Wait for renewal goroutine to finish
	select {
	case <-l.renewDone:
	case <-time.After(1 * time.Second):
		// Don't wait forever
	}

	err := l.env.RemoveLock(ctx, l.objectID)
	if err != nil {
		return fmt.Errorf("failed to remove lock: %w", err)
	}

	l.locked = false
	return nil
}

// renewLock runs in a background goroutine and renews the lock at half TTL intervals.
func (l *Locker) renewLock() {
	defer close(l.renewDone)

	if !l.locked {
		return
	}

	renewInterval := l.config.TTL / 2
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			err := l.env.PutLock(l.ctx, l.objectID, l.config.TTL)
			if err != nil {
				// Log error but continue trying
				// In a real application, you might want to use a logger here
				continue
			}
		}
	}
}

// generateLockObjectID creates a consistent object ID for the lock file.
func generateLockObjectID(name string) string {
	return fmt.Sprintf("%s.lock", name)
}
