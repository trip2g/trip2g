package cleanupwebhookdeliverylogs

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
)

type Env interface {
	CleanupOldDeliveryLogs(ctx context.Context) error
	Logger() logger.Logger
}

// Result holds cleanup statistics.
type Result struct {
	Cleaned bool
}

// Resolve deletes webhook delivery logs older than 7 days.
func Resolve(ctx context.Context, env Env) (*Result, error) {
	err := env.CleanupOldDeliveryLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup old delivery logs: %w", err)
	}

	return &Result{Cleaned: true}, nil
}
