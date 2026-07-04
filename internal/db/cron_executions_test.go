package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

// TestUpdateRunningCronJobExecutions_MarksRowDead is a regression test for the
// "mark stale running executions as dead on startup" feature.
//
// Root cause: the SQL WHERE clause used status = 'running' (a string literal)
// but the status column is INTEGER (running=1). SQLite coerces 'running' to 0,
// so the predicate never matched and the update was a permanent no-op.
//
// RED: on unfixed code the row stays status=1 (running) after the call.
// GREEN: after the fix the row is status=3 (failed) with the "died" message.
func TestUpdateRunningCronJobExecutions_MarksRowDead(t *testing.T) {
	conn, _, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	wq := db.NewWriteQueries(conn)

	// Insert a cron_jobs row (required by the FK on cron_job_executions).
	var jobID int64
	err := conn.QueryRow(`
		insert into cron_jobs (name, expression)
		values ('test-job', '0 * * * *')
		returning id
	`).Scan(&jobID)
	require.NoError(t, err)

	// Insert a cron_job_executions row with status = 1 (running).
	// This simulates a row left behind by a killed/crashed instance.
	const statusRunning int64 = 1
	var execID int64
	err = conn.QueryRow(`
		insert into cron_job_executions (job_id, status)
		values (?, ?)
		returning id
	`, jobID, statusRunning).Scan(&execID)
	require.NoError(t, err)

	// Call the mark-dead path: WHERE status = running → SET status = failed.
	const statusFailed int64 = 3
	died := "died"
	updateErr := wq.UpdateRunningCronJobExecutions(ctx, db.UpdateRunningCronJobExecutionsParams{
		JobID:        jobID,
		Status:       statusFailed,
		ErrorMessage: &died,
	})
	require.NoError(t, updateErr)

	// Assert: the row must now be status=3 (failed) with the error message.
	var gotStatus int64
	var gotMsg *string
	err = conn.QueryRow(
		`select status, error_message from cron_job_executions where id = ?`,
		execID,
	).Scan(&gotStatus, &gotMsg)
	require.NoError(t, err)

	require.Equal(t, statusFailed, gotStatus,
		"stale running execution must be marked failed (dead); got status=%d — "+
			"likely the WHERE status='running' string comparison bug is still present", gotStatus)
	require.NotNil(t, gotMsg)
	require.Equal(t, "died", *gotMsg)
}
