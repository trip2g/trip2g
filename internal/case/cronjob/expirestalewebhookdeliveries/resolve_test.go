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
		ExpireStaleWebhookDeliveriesFunc:     func(_ context.Context, _ string) error { return nil },
		ExpireStaleCronWebhookDeliveriesFunc: func(_ context.Context, _ string) error { return nil },
		AgentDeliveryCooldownSecondsFunc:     func() int { return 60 },
		LoggerFunc:                           func() logger.Logger { return &logger.DummyLogger{} },
	}
}

func TestResolve_ExpiresBothTables(t *testing.T) {
	var changeWin, cronWin string
	env := newEnv()
	env.ExpireStaleWebhookDeliveriesFunc = func(_ context.Context, w string) error { changeWin = w; return nil }
	env.ExpireStaleCronWebhookDeliveriesFunc = func(_ context.Context, w string) error { cronWin = w; return nil }

	_, err := expirestalewebhookdeliveries.Resolve(context.Background(), env)
	require.NoError(t, err)
	require.Equal(t, "-60 seconds", changeWin)
	require.Equal(t, "-60 seconds", cronWin)
}

func TestResolve_PropagatesError(t *testing.T) {
	env := newEnv()
	env.ExpireStaleWebhookDeliveriesFunc = func(_ context.Context, _ string) error { return errors.New("boom") }
	_, err := expirestalewebhookdeliveries.Resolve(context.Background(), env)
	require.Error(t, err)
}
