package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

// insertTestChangeWebhook creates a minimal change_webhook row with the given timeout.
func insertTestChangeWebhook(t *testing.T, conn *sql.DB, adminID int64, timeoutSeconds int) int64 {
	t.Helper()
	var id int64
	err := conn.QueryRow(`
		insert into change_webhooks
		  (url, include_patterns, secret, timeout_seconds, created_by)
		values ('https://example.com/hook', '["**"]', 'secret', ?, ?)
		returning id
	`, timeoutSeconds, adminID).Scan(&id)
	require.NoError(t, err)
	return id
}

// insertTestCronWebhook creates a minimal cron_webhook row with the given timeout.
func insertTestCronWebhook(t *testing.T, conn *sql.DB, adminID int64, timeoutSeconds int) int64 {
	t.Helper()
	var id int64
	err := conn.QueryRow(`
		insert into cron_webhooks
		  (url, cron_schedule, secret, timeout_seconds, created_by)
		values ('https://example.com/cronhook', '0 * * * *', 'secret', ?, ?)
		returning id
	`, timeoutSeconds, adminID).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestExpireStaleWebhookDeliveries_PerWebhookTimeout is a behavioral regression test
// for F7: the janitor must use each webhook's own timeout_seconds+30 margin when
// deciding which deliveries are stale. It must NOT use a global cooldown constant.
//
// Covered cases:
//  1. Young delivery (within timeout+margin): NOT reaped, skip mode still blocks re-insert.
//  2. Old delivery (past timeout+margin): reaped to 'failed'.
//  3. Long delivery (>global 60s cooldown, <per-webhook 3600s timeout): NOT reaped.
//  4. Stale 'pending' delivery (past timeout+margin): reaped to 'failed'.
func TestExpireStaleWebhookDeliveries_PerWebhookTimeout(t *testing.T) {
	conn, _, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	wq := db.NewWriteQueries(conn)
	adminID := insertTestAdminUser(t, conn)

	// Case 1: Young 'running' delivery within timeout+margin must NOT be reaped.
	// Webhook timeout = 300s. Delivery created now → (now + 330s) > now → not stale.
	whYoung := insertTestChangeWebhook(t, conn, adminID, 300)
	var youngID int64
	require.NoError(t, conn.QueryRow(
		`insert into change_webhook_deliveries (webhook_id, status) values (?, 'running') returning id`,
		whYoung,
	).Scan(&youngID))

	// Case 2: Old 'running' delivery past timeout+margin must be reaped.
	// Webhook timeout = 60s (90s window). Delivery created 2 h ago → stale.
	whOld := insertTestChangeWebhook(t, conn, adminID, 60)
	var oldID int64
	require.NoError(t, conn.QueryRow(
		`insert into change_webhook_deliveries (webhook_id, status) values (?, 'running') returning id`,
		whOld,
	).Scan(&oldID))
	mustExec(t, conn,
		`update change_webhook_deliveries set created_at = datetime('now', '-7200 seconds') where id = ?`,
		oldID)

	// Case 3: Long 'running' delivery past the old global 60s cooldown but within
	// per-webhook 3600s timeout must NOT be reaped.
	// 120s old, timeout=3600s → window = 3630s → (now-120)+3630 = now+3510 > now.
	whLong := insertTestChangeWebhook(t, conn, adminID, 3600)
	var longID int64
	require.NoError(t, conn.QueryRow(
		`insert into change_webhook_deliveries (webhook_id, status) values (?, 'running') returning id`,
		whLong,
	).Scan(&longID))
	mustExec(t, conn,
		`update change_webhook_deliveries set created_at = datetime('now', '-120 seconds') where id = ?`,
		longID)

	// Case 4: Stale 'pending' delivery past timeout+margin must also be reaped.
	// Webhook timeout = 60s. Delivery created 2 h ago, status='pending' → stale.
	whPending := insertTestChangeWebhook(t, conn, adminID, 60)
	var pendingID int64
	require.NoError(t, conn.QueryRow(
		`insert into change_webhook_deliveries (webhook_id, status) values (?, 'pending') returning id`,
		whPending,
	).Scan(&pendingID))
	mustExec(t, conn,
		`update change_webhook_deliveries set created_at = datetime('now', '-7200 seconds') where id = ?`,
		pendingID)

	// Run the janitor.
	require.NoError(t, wq.ExpireStaleWebhookDeliveries(ctx))

	checkStatus := func(t *testing.T, id int64) string {
		t.Helper()
		var status string
		require.NoError(t, conn.QueryRow(
			`select status from change_webhook_deliveries where id = ?`, id,
		).Scan(&status))
		return status
	}

	require.Equal(t, "running", checkStatus(t, youngID),
		"young delivery within timeout window must not be reaped")
	require.Equal(t, "failed", checkStatus(t, oldID),
		"old delivery past timeout+margin must be reaped to failed")
	require.Equal(t, "running", checkStatus(t, longID),
		"delivery past global 60s cooldown but within per-webhook 3600s timeout must not be reaped")
	require.Equal(t, "failed", checkStatus(t, pendingID),
		"stale pending delivery past timeout+margin must be reaped to failed")
}

// TestExpireStaleCronWebhookDeliveries_PerWebhookTimeout mirrors the change-webhook
// janitor test but for cron_webhook_deliveries / ExpireStaleCronWebhookDeliveries.
func TestExpireStaleCronWebhookDeliveries_PerWebhookTimeout(t *testing.T) {
	conn, _, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	wq := db.NewWriteQueries(conn)
	adminID := insertTestAdminUser(t, conn)

	// Case 1: Young 'running' delivery within timeout+margin must NOT be reaped.
	whYoung := insertTestCronWebhook(t, conn, adminID, 300)
	var youngID int64
	require.NoError(t, conn.QueryRow(
		`insert into cron_webhook_deliveries (cron_webhook_id, status) values (?, 'running') returning id`,
		whYoung,
	).Scan(&youngID))

	// Case 2: Old 'running' delivery past timeout+margin must be reaped.
	whOld := insertTestCronWebhook(t, conn, adminID, 60)
	var oldID int64
	require.NoError(t, conn.QueryRow(
		`insert into cron_webhook_deliveries (cron_webhook_id, status) values (?, 'running') returning id`,
		whOld,
	).Scan(&oldID))
	mustExec(t, conn,
		`update cron_webhook_deliveries set created_at = datetime('now', '-7200 seconds') where id = ?`,
		oldID)

	// Case 3: Long 'running' delivery past global 60s but within per-webhook 3600s timeout.
	whLong := insertTestCronWebhook(t, conn, adminID, 3600)
	var longID int64
	require.NoError(t, conn.QueryRow(
		`insert into cron_webhook_deliveries (cron_webhook_id, status) values (?, 'running') returning id`,
		whLong,
	).Scan(&longID))
	mustExec(t, conn,
		`update cron_webhook_deliveries set created_at = datetime('now', '-120 seconds') where id = ?`,
		longID)

	// Case 4: Stale 'pending' delivery past timeout+margin must also be reaped.
	whPending := insertTestCronWebhook(t, conn, adminID, 60)
	var pendingID int64
	require.NoError(t, conn.QueryRow(
		`insert into cron_webhook_deliveries (cron_webhook_id, status) values (?, 'pending') returning id`,
		whPending,
	).Scan(&pendingID))
	mustExec(t, conn,
		`update cron_webhook_deliveries set created_at = datetime('now', '-7200 seconds') where id = ?`,
		pendingID)

	// Run the janitor.
	require.NoError(t, wq.ExpireStaleCronWebhookDeliveries(ctx))

	checkStatus := func(t *testing.T, id int64) string {
		t.Helper()
		var status string
		require.NoError(t, conn.QueryRow(
			`select status from cron_webhook_deliveries where id = ?`, id,
		).Scan(&status))
		return status
	}

	require.Equal(t, "running", checkStatus(t, youngID),
		"young cron delivery within timeout window must not be reaped")
	require.Equal(t, "failed", checkStatus(t, oldID),
		"old cron delivery past timeout+margin must be reaped to failed")
	require.Equal(t, "running", checkStatus(t, longID),
		"cron delivery past global 60s cooldown but within per-webhook 3600s timeout must not be reaped")
	require.Equal(t, "failed", checkStatus(t, pendingID),
		"stale pending cron delivery past timeout+margin must be reaped to failed")
}
