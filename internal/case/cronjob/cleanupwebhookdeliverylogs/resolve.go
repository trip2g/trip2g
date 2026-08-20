package cleanupwebhookdeliverylogs

import (
	"context"
	"fmt"
	"time"

	"trip2g/internal/logger"
)

type Env interface {
	CleanupOldDeliveryLogs(ctx context.Context, cutoffTime time.Time) error
	Logger() logger.Logger
}

// Result holds cleanup statistics.
type Result struct {
	Cleaned bool
}

// Resolve deletes webhook delivery logs older than the configured retention period.
func Resolve(ctx context.Context, env Env, cfg Config) (*Result, error) {
	cutoff := time.Now().Add(-cfg.Retention)

	err := env.CleanupOldDeliveryLogs(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup old delivery logs: %w", err)
	}

	return &Result{Cleaned: true}, nil
}
