package expirestalewebhookdeliveries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/cronjob/expirestalewebhookdeliveries"
	"trip2g/internal/logger"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg expirestalewebhookdeliveries_test . Env

func newEnv() *EnvMock {
	return &EnvMock{
		ExpireStaleWebhookDeliveriesFunc:     func(_ context.Context) error { return nil },
		ExpireStaleCronWebhookDeliveriesFunc: func(_ context.Context) error { return nil },
		LoggerFunc:                           func() logger.Logger { return &logger.DummyLogger{} },
	}
}

// TestResolve_ExpiresBothTables verifies that Resolve calls both expire methods.
func TestResolve_ExpiresBothTables(t *testing.T) {
	var changeExpired, cronExpired bool
	env := newEnv()
	env.ExpireStaleWebhookDeliveriesFunc = func(_ context.Context) error { changeExpired = true; return nil }
	env.ExpireStaleCronWebhookDeliveriesFunc = func(_ context.Context) error { cronExpired = true; return nil }

	_, err := expirestalewebhookdeliveries.Resolve(context.Background(), env)
	require.NoError(t, err)
	require.True(t, changeExpired, "change webhook deliveries must be expired")
	require.True(t, cronExpired, "cron webhook deliveries must be expired")
}

// TestResolve_PropagatesError verifies that an error from ExpireStaleWebhookDeliveries
// is surfaced to the caller.
func TestResolve_PropagatesError(t *testing.T) {
	env := newEnv()
	env.ExpireStaleWebhookDeliveriesFunc = func(_ context.Context) error { return errors.New("boom") }
	_, err := expirestalewebhookdeliveries.Resolve(context.Background(), env)
	require.Error(t, err)
}

// TestResolve_JanitorDoesNotPassGlobalStaleWindow is a regression test for F7:
// the janitor must NOT pass a global stale window to the expire functions.
// Instead, per-webhook timeout_seconds is used directly in the SQL (JOIN logic).
// Before the fix the Env interface had a staleWindow string parameter; after the fix
// the parameter is gone and staleness is computed per-row in SQL.
func TestResolve_JanitorDoesNotPassGlobalStaleWindow(t *testing.T) {
	// Both expire functions must be called exactly once with only a context.
	// The absence of a staleWindow parameter in the interface IS the fix.
	env := newEnv()
	_, err := expirestalewebhookdeliveries.Resolve(context.Background(), env)
	require.NoError(t, err)

	require.Len(t, env.ExpireStaleWebhookDeliveriesCalls(), 1,
		"ExpireStaleWebhookDeliveries must be called exactly once")
	require.Len(t, env.ExpireStaleCronWebhookDeliveriesCalls(), 1,
		"ExpireStaleCronWebhookDeliveries must be called exactly once")
}
