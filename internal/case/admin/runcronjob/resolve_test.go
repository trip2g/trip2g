package runcronjob

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/cronjobs"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

type stubEnv struct {
	executeFn func(jobID int64) (*db.CronJobExecution, error)
}

func (e *stubEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{}, nil
}

func (e *stubEnv) ExecuteCronJobManually(jobID int64) (*db.CronJobExecution, error) {
	return e.executeFn(jobID)
}

func (e *stubEnv) Logger() logger.Logger { return &logger.DummyLogger{} }

// A duplicate manual run must surface a clear "already running" message, never
// a synthetic execution row (the in-memory dedup placeholder has ID 0).
func TestResolve_AlreadyRunningReturnsErrorPayload(t *testing.T) {
	env := &stubEnv{
		executeFn: func(_ int64) (*db.CronJobExecution, error) {
			return nil, cronjobs.ErrJobAlreadyRunning
		},
	}

	payload, err := Resolve(context.Background(), env, Input{ID: 1})
	require.NoError(t, err)

	errPayload, ok := payload.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload, got %T", payload)
	require.Equal(t, "Cron job is already running", errPayload.Message)
}
