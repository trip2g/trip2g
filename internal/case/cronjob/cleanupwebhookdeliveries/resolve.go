package cleanupwebhookdeliveries

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
)

type Env interface {
	CleanupOldChangeWebhookDeliveries(ctx context.Context) error
	CleanupOldCronWebhookDeliveries(ctx context.Context) error
	Logger() logger.Logger
}

// Result holds cleanup statistics.
type Result struct {
	Cleaned bool
}

// Resolve deletes webhook deliveries older than 30 days.
func Resolve(ctx context.Context, env Env) (*Result, error) {
	err := env.CleanupOldChangeWebhookDeliveries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup old change webhook deliveries: %w", err)
	}

	err = env.CleanupOldCronWebhookDeliveries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup old cron webhook deliveries: %w", err)
	}

	return &Result{Cleaned: true}, nil
}
