package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

// TestUpdateWebhookDeliveryResult_SpendCoalesce verifies that a status-only
// UpdateWebhookDeliveryResult call (nil Costs) does not clobber the cost object
// written by a prior call.
func TestUpdateWebhookDeliveryResult_SpendCoalesce(t *testing.T) {
	conn, _, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	wq := db.NewWriteQueries(conn)
	adminID := insertTestAdminUser(t, conn)

	whID := insertTestChangeWebhook(t, conn, adminID, 60)

	del, err := wq.InsertWebhookDelivery(ctx, db.InsertWebhookDeliveryParams{
		WebhookID: whID,
		Attempt:   1,
	})
	require.NoError(t, err)

	// First update: write spend.
	costs := `{"tokens":123,"steps":5}`
	require.NoError(t, wq.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: nil,
		DurationMs:     nil,
		Costs:          &costs,
		ID:             del.ID,
	}))

	// Second update: status-only, nil costs → must NOT overwrite what was reported.
	require.NoError(t, wq.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "failed",
		ResponseStatus: nil,
		DurationMs:     nil,
		Costs:          nil,
		ID:             del.ID,
	}))

	var gotCosts *string
	require.NoError(t, conn.QueryRowContext(ctx,
		`select costs from change_webhook_deliveries where id = ?`, del.ID,
	).Scan(&gotCosts))

	require.NotNil(t, gotCosts, "costs must not be clobbered by a status-only update")
	require.JSONEq(t, `{"tokens":123,"steps":5}`, *gotCosts)
}

// TestUpdateCronWebhookDeliveryResult_SpendCoalesce is the cron-twin of the
// change-webhook spend-coalesce test.
func TestUpdateCronWebhookDeliveryResult_SpendCoalesce(t *testing.T) {
	conn, _, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	wq := db.NewWriteQueries(conn)
	adminID := insertTestAdminUser(t, conn)

	cronWhID := insertTestCronWebhook(t, conn, adminID, 60)

	del, err := wq.InsertCronWebhookDelivery(ctx, db.InsertCronWebhookDeliveryParams{
		CronWebhookID: cronWhID,
		Attempt:       1,
	})
	require.NoError(t, err)

	// First update: write spend.
	costs := `{"tokens":123,"steps":5}`
	require.NoError(t, wq.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: nil,
		DurationMs:     nil,
		Costs:          &costs,
		ID:             del.ID,
	}))

	// Second update: status-only, nil costs → must NOT overwrite what was reported.
	require.NoError(t, wq.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
		Status:         "failed",
		ResponseStatus: nil,
		DurationMs:     nil,
		Costs:          nil,
		ID:             del.ID,
	}))

	var gotCosts *string
	require.NoError(t, conn.QueryRowContext(ctx,
		`select costs from cron_webhook_deliveries where id = ?`, del.ID,
	).Scan(&gotCosts))

	require.NotNil(t, gotCosts, "costs must not be clobbered by a status-only update")
	require.JSONEq(t, `{"tokens":123,"steps":5}`, *gotCosts)
}
