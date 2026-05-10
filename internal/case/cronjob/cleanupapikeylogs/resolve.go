package cleanupapikeylogs

import (
	"context"
	"fmt"
	"time"

	"trip2g/internal/logger"
)

type Env interface {
	CleanupOldAPIKeyLogs(ctx context.Context, cutoffTime time.Time) error
	CleanupOrphanedAPIKeyLogIPs(ctx context.Context) error
	Logger() logger.Logger
}

// Result holds cleanup statistics.
type Result struct {
	Cleaned bool
}

// Resolve deletes API key logs older than the configured retention period,
// then removes orphaned IP entries from the normalization table.
func Resolve(ctx context.Context, env Env, cfg Config) (*Result, error) {
	cutoff := time.Now().Add(-cfg.Retention)

	if err := env.CleanupOldAPIKeyLogs(ctx, cutoff); err != nil {
		return nil, fmt.Errorf("failed to cleanup old api key logs: %w", err)
	}

	if err := env.CleanupOrphanedAPIKeyLogIPs(ctx); err != nil {
		return nil, fmt.Errorf("failed to cleanup orphaned api key log ips: %w", err)
	}

	return &Result{Cleaned: true}, nil
}
