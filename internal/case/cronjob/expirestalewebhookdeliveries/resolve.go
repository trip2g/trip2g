package expirestalewebhookdeliveries

import (
	"context"
	"fmt"

	"trip2g/internal/logger"
)

type Env interface {
	ExpireStaleWebhookDeliveries(ctx context.Context, staleWindow string) error
	ExpireStaleCronWebhookDeliveries(ctx context.Context, staleWindow string) error
	AgentDeliveryCooldownSeconds() int
	Logger() logger.Logger
}

// Result reports whether the sweep ran.
type Result struct {
	Expired bool
}

// Resolve finalizes orphaned 'running' webhook deliveries (change + cron) whose
// liveness window has lapsed, marking them 'failed'.
func Resolve(ctx context.Context, env Env) (*Result, error) {
	staleWindow := fmt.Sprintf("-%d seconds", env.AgentDeliveryCooldownSeconds())

	if err := env.ExpireStaleWebhookDeliveries(ctx, staleWindow); err != nil {
		return nil, fmt.Errorf("failed to expire stale change webhook deliveries: %w", err)
	}
	if err := env.ExpireStaleCronWebhookDeliveries(ctx, staleWindow); err != nil {
		return nil, fmt.Errorf("failed to expire stale cron webhook deliveries: %w", err)
	}

	return &Result{Expired: true}, nil
}
