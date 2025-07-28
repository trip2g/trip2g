package patreonjobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"trip2g/internal/case/refreshpatreondata"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger

	refreshpatreondata.Env
}

type PatreonJobs struct {
	env Env
	mu  sync.Mutex

	cancelMap map[int64]context.CancelFunc
}

func New(ctx context.Context, env Env) (*PatreonJobs, error) {
	io := PatreonJobs{
		env:       env,
		cancelMap: make(map[int64]context.CancelFunc),
	}

	credentials, err := env.AllActivePatreonCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active Patreon credentials: %w", err)
	}

	for _, cred := range credentials {
		err := io.StartPatreonRefreshBackgroundJob(ctx, cred.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to start Patreon refresh background job for credentials ID %d: %w", cred.ID, err)
		}
	}

	return &io, nil
}

func (io *PatreonJobs) StartPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64) error {
	io.env.Logger().Info("starting Patreon refresh background job", "credentialsID", credentialsID)

	ctx, cancel := context.WithCancel(ctx)

	io.mu.Lock()
	defer io.mu.Unlock()

	existingCancel, exists := io.cancelMap[credentialsID]
	if exists {
		existingCancel()
	}

	io.cancelMap[credentialsID] = cancel

	go func() {
		// 1 hour timer
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Call the refresh function
				err := refreshpatreondata.Resolve(ctx, io.env, &credentialsID)
				if err != nil {
					io.env.Logger().Error("failed to refresh Patreon data", "credentialsID", credentialsID, "error", err)
				}
			case <-ctx.Done():
				io.env.Logger().Info("Patreon refresh background job stopped", "credentialsID", credentialsID)
				return
			}
		}
	}()

	return nil
}

func (io *PatreonJobs) StopPatreonRefreshBackgroundJob(ctx context.Context, credentialsID int64) error {
	io.env.Logger().Info("stopping Patreon refresh background job", "credentialsID", credentialsID)

	io.mu.Lock()
	defer io.mu.Unlock()

	cancelFunc, exists := io.cancelMap[credentialsID]
	if exists {
		cancelFunc()
	}

	return nil
}
