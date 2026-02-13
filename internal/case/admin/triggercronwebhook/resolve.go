package triggercronwebhook

import (
	"context"
	"fmt"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	CronWebhookByID(ctx context.Context, id int64) (db.CronWebhook, error)
	InsertCronWebhookDelivery(ctx context.Context, arg db.InsertCronWebhookDeliveryParams) (db.CronWebhookDelivery, error)
	EnqueueDeliverCronWebhook(ctx context.Context, params delivercronwebhook.DeliverCronParams) error
}

// Resolve manually triggers a cron webhook by creating a delivery and enqueuing the job.
func Resolve(ctx context.Context, env Env, input model.TriggerCronWebhookInput) (model.TriggerCronWebhookOrErrorPayload, error) {
	// Check admin authorization.
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	// Load the webhook to ensure it exists.
	webhook, err := env.CronWebhookByID(ctx, input.CronWebhookID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cron webhook: %w", err)
	}

	// Check if webhook is enabled.
	if !webhook.Enabled {
		return &model.ErrorPayload{
			Message:  "Cannot trigger disabled webhook",
			ByFields: nil,
		}, nil
	}

	// Create delivery record.
	delivery, err := env.InsertCronWebhookDelivery(ctx, db.InsertCronWebhookDeliveryParams{
		CronWebhookID: webhook.ID,
		Attempt:       1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert cron webhook delivery: %w", err)
	}

	// Enqueue background job.
	err = env.EnqueueDeliverCronWebhook(ctx, delivercronwebhook.DeliverCronParams{
		DeliveryID:    delivery.ID,
		CronWebhookID: webhook.ID,
		Attempt:       1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue cron webhook delivery: %w", err)
	}

	return &model.TriggerCronWebhookPayload{
		DeliveryID: delivery.ID,
	}, nil
}
