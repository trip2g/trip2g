package boostyjobs

import (
	"context"
	"fmt"
	"sync"
	"time"
	"trip2g/internal/case/refreshboostydata"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type Config struct {
	RefreshInterval time.Duration // How often to refresh data (default: 1 hour)
	ImmediatelyGap  time.Duration // How old synced_at must be to trigger immediate refresh (default: 10 minutes)
}

func DefaultConfig() Config {
	return Config{
		RefreshInterval: 1 * time.Hour,
		ImmediatelyGap:  10 * time.Minute,
	}
}

type Env interface {
	Logger() logger.Logger
	AllActiveBoostyCredentials(ctx context.Context) ([]db.BoostyCredential, error)

	refreshboostydata.Env
}

type BoostyJobs struct {
	mu     sync.Mutex
	env    Env
	config Config

	cancelMap map[int64]context.CancelFunc // maps credential ID to cancel function
	logger    logger.Logger
}

func New(ctx context.Context, env Env, config Config) (*BoostyJobs, error) {
	io := BoostyJobs{
		env:    env,
		config: config,
		logger: logger.WithPrefix(env.Logger(), "boostyjobs:"),

		cancelMap: make(map[int64]context.CancelFunc),
	}

	credentials, err := env.AllActiveBoostyCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active Boosty credentials: %w", err)
	}

	for _, cred := range credentials {
		// Check if credentials need immediate refresh based on config
		immediately := false
		if cred.SyncedAt == nil || time.Since(*cred.SyncedAt) > io.config.ImmediatelyGap {
			immediately = true
			var lastSync time.Time
			if cred.SyncedAt != nil {
				lastSync = *cred.SyncedAt
			}
			io.logger.Info("credentials need immediate refresh", "credentialsID", cred.ID, "lastSync", lastSync, "gap", io.config.ImmediatelyGap)
		}

		startErr := io.StartBoostyRefreshBackgroundJob(ctx, cred.ID, immediately)
		if startErr != nil {
			io.logger.Error("failed to start Boosty refresh background job", "credentialsID", cred.ID, "error", startErr)
		}
	}

	// refresh token
	// update members & tears

	return &io, nil
}

func (io *BoostyJobs) StartBoostyRefreshBackgroundJob(ctx context.Context, credentialsID int64, immediately bool) error {
	io.mu.Lock()
	defer io.mu.Unlock()

	existingCancel, exists := io.cancelMap[credentialsID]
	if exists {
		existingCancel()
	}

	ctx, cancel := context.WithCancel(ctx)

	io.cancelMap[credentialsID] = cancel

	io.logger.Info("starting Boosty refresh background job", "credentialsID", credentialsID)

	go func() {
		// Run immediately if requested
		if immediately {
			err := refreshboostydata.Resolve(ctx, io.env, credentialsID)
			if err != nil {
				io.logger.Error("failed to refresh Boosty data (immediate)", "credentialID", credentialsID, "error", err)
			} else {
				io.logger.Info("successfully refreshed Boosty data (immediate)", "credentialID", credentialsID)
			}
		}

		// Timer based on config
		ticker := time.NewTicker(io.config.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := refreshboostydata.Resolve(ctx, io.env, credentialsID)
				if err != nil {
					io.logger.Error("failed to refresh Boosty data", "credentialID", credentialsID, "error", err)
				} else {
					io.logger.Info("successfully refreshed Boosty data", "credentialID", credentialsID)
				}

			case <-ctx.Done():
				io.logger.Info("Boosty refresh job cancelled", "credentialID", credentialsID)
				return
			}
		}
	}()

	return nil
}

func (io *BoostyJobs) StopBoostyRefreshBackgroundJob(ctx context.Context, credentialsID int64) error {
	io.logger.Info("stopping Boosty refresh background job", "credentialsID", credentialsID)

	io.mu.Lock()
	defer io.mu.Unlock()

	cancelFunc, exists := io.cancelMap[credentialsID]
	if exists {
		cancelFunc()
		delete(io.cancelMap, credentialsID)
	}

	return nil
}
