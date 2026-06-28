package expirestalewebhookdeliveries

import (
	"context"
	"fmt"

	"trip2g/internal/logger"
)

type Env interface {
	// ExpireStaleWebhookDeliveries finalizes orphaned 'running'/'pending' change webhook
	// deliveries whose per-webhook liveness window (timeout_seconds + margin) has lapsed.
	ExpireStaleWebhookDeliveries(ctx context.Context) error
	// ExpireStaleCronWebhookDeliveries finalizes orphaned 'running'/'pending' cron webhook
	// deliveries whose per-webhook liveness window (timeout_seconds + margin) has lapsed.
	ExpireStaleCronWebhookDeliveries(ctx context.Context) error
	Logger() logger.Logger
}

// Result reports whether the sweep ran.
type Result struct {
	Expired bool
}

// Resolve finalizes orphaned 'running'/'pending' webhook deliveries (change + cron)
// whose liveness window has lapsed, marking them 'failed'.
// Staleness is determined per-webhook using each webhook's timeout_seconds (+ a
// 30-second margin) so long-running agent deliveries are not reaped prematurely.
func Resolve(ctx context.Context, env Env) (*Result, error) {
	if err := env.ExpireStaleWebhookDeliveries(ctx); err != nil {
		return nil, fmt.Errorf("failed to expire stale change webhook deliveries: %w", err)
	}
	if err := env.ExpireStaleCronWebhookDeliveries(ctx); err != nil {
		return nil, fmt.Errorf("failed to expire stale cron webhook deliveries: %w", err)
	}

	return &Result{Expired: true}, nil
}
