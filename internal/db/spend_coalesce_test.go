package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

// TestUpdateWebhookDeliveryResult_SpendCoalesce verifies that a status-only
// UpdateWebhookDeliveryResult call (nil TokensUsed/Steps) does not clobber
// spend values written by a prior call.
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
	tokensUsed := int64(123)
	steps := int64(5)
	require.NoError(t, wq.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: nil,
		DurationMs:     nil,
		TokensUsed:     &tokensUsed,
		Steps:          &steps,
		ID:             del.ID,
	}))

	// Second update: status-only, nil spend → must NOT overwrite tokens_used/steps.
	require.NoError(t, wq.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "failed",
		ResponseStatus: nil,
		DurationMs:     nil,
		TokensUsed:     nil,
		Steps:          nil,
		ID:             del.ID,
	}))

	var gotTokens, gotSteps *int64
	require.NoError(t, conn.QueryRowContext(ctx,
		`select tokens_used, steps from change_webhook_deliveries where id = ?`, del.ID,
	).Scan(&gotTokens, &gotSteps))

	require.NotNil(t, gotTokens, "tokens_used must not be clobbered by status-only update")
	require.EqualValues(t, 123, *gotTokens)
	require.NotNil(t, gotSteps, "steps must not be clobbered by status-only update")
	require.EqualValues(t, 5, *gotSteps)
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
	tokensUsed := int64(123)
	steps := int64(5)
	require.NoError(t, wq.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: nil,
		DurationMs:     nil,
		TokensUsed:     &tokensUsed,
		Steps:          &steps,
		ID:             del.ID,
	}))

	// Second update: status-only, nil spend → must NOT overwrite tokens_used/steps.
	require.NoError(t, wq.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
		Status:         "failed",
		ResponseStatus: nil,
		DurationMs:     nil,
		TokensUsed:     nil,
		Steps:          nil,
		ID:             del.ID,
	}))

	var gotTokens, gotSteps *int64
	require.NoError(t, conn.QueryRowContext(ctx,
		`select tokens_used, steps from cron_webhook_deliveries where id = ?`, del.ID,
	).Scan(&gotTokens, &gotSteps))

	require.NotNil(t, gotTokens, "tokens_used must not be clobbered by status-only update")
	require.EqualValues(t, 123, *gotTokens)
	require.NotNil(t, gotSteps, "steps must not be clobbered by status-only update")
	require.EqualValues(t, 5, *gotSteps)
}
