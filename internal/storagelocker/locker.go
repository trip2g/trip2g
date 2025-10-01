package storagelocker

import (
	"context"
	"errors"
	"fmt"
)

type Env interface {
	PutLock(ctx context.Context, objectID string) error
	RemoveLock(ctx context.Context, objectID string) error
}

type Config struct {
	Enabled bool
	Name    string
}

type Locker struct {
	config   Config
	env      Env
	objectID string
	locked   bool
}

// DefaultConfig returns a default configuration for the storage locker.
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Name:    "storage",
	}
}

var ErrStorageNotSupportLock = errors.New("storage does not support locking")

// New creates a new storage locker and immediately acquires the lock if enabled.
// It panics if the lock already exists, ensuring only one instance can run.
// If disabled, returns a no-op locker.
func New(ctx context.Context, config Config, env Env) (*Locker, error) {
	locker := &Locker{
		config: config,
		env:    env,
		locked: false,
	}

	if !config.Enabled {
		return locker, nil
	}

	objectID := generateLockObjectID(config.Name)

	err := env.PutLock(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("storage lock already exists or failed to acquire: %w", err)
	}

	// Check if the storage supports locking by attempting to put the lock again.
	err = env.PutLock(ctx, objectID)
	if err == nil {
		return nil, ErrStorageNotSupportLock
	}

	locker.objectID = objectID
	locker.locked = true

	return locker, nil
}

// Unlock removes the lock file for graceful shutdown.
func (l *Locker) Unlock(ctx context.Context) error {
	if !l.locked {
		return nil
	}

	err := l.env.RemoveLock(ctx, l.objectID)
	if err != nil {
		return fmt.Errorf("failed to remove lock: %w", err)
	}

	l.locked = false
	return nil
}

// generateLockObjectID creates a consistent object ID for the lock file.
func generateLockObjectID(name string) string {
	return fmt.Sprintf("%s.lock", name)
}
