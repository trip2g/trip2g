package cleanupwebhookdeliveries

import (
	"context"
	"fmt"
	"time"

	"trip2g/internal/logger"
)

type Env interface {
	CleanupOldChangeWebhookDeliveries(ctx context.Context, cutoffTime time.Time) error
	CleanupOldCronWebhookDeliveries(ctx context.Context, cutoffTime time.Time) error
	Logger() logger.Logger
}

// Result holds cleanup statistics.
type Result struct {
	Cleaned bool
}

// Resolve deletes webhook deliveries older than the configured retention period.
func Resolve(ctx context.Context, env Env, cfg Config) (*Result, error) {
	cutoff := time.Now().Add(-cfg.Retention)

	err := env.CleanupOldChangeWebhookDeliveries(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup old change webhook deliveries: %w", err)
	}

	err = env.CleanupOldCronWebhookDeliveries(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup old cron webhook deliveries: %w", err)
	}

	return &Result{Cleaned: true}, nil
}
