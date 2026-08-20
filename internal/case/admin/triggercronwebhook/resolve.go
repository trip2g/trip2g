package triggercronwebhook

import (
	"context"
	"fmt"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	CronWebhookByID(ctx context.Context, id int64) (db.CronWebhook, error)
	InsertCronWebhookDelivery(ctx context.Context, arg db.InsertCronWebhookDeliveryParams) (db.CronWebhookDelivery, error)
	SetCronWebhookDeliveryChain(ctx context.Context, arg db.SetCronWebhookDeliveryChainParams) error
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

	// A manual run is a chain root, same as a scheduled one.
	trace := webhookutil.TraceID(webhookutil.DeliveryKindCron, delivery.ID)
	chainErr := env.SetCronWebhookDeliveryChain(ctx, db.SetCronWebhookDeliveryChainParams{
		ID:    delivery.ID,
		Trace: ptr.To(trace),
	})
	if chainErr != nil {
		return nil, fmt.Errorf("failed to stamp cron delivery chain: %w", chainErr)
	}

	// Enqueue background job.
	err = env.EnqueueDeliverCronWebhook(ctx, delivercronwebhook.DeliverCronParams{
		DeliveryID:    delivery.ID,
		CronWebhookID: webhook.ID,
		Attempt:       1,
		Trace:         trace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue cron webhook delivery: %w", err)
	}

	return &model.TriggerCronWebhookPayload{
		DeliveryID: delivery.ID,
	}, nil
}
