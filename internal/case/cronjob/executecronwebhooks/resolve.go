package executecronwebhooks

import (
	"context"
	"fmt"
	"time"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/robfig/cron/v3"
)

type Env interface {
	ListCronWebhooksDueForExecution(ctx context.Context) ([]db.CronWebhook, error)
	UpdateCronWebhookNextRunAt(ctx context.Context, arg db.UpdateCronWebhookNextRunAtParams) error
	InsertCronWebhookDelivery(ctx context.Context, arg db.InsertCronWebhookDeliveryParams) (db.CronWebhookDelivery, error)
	EnqueueDeliverCronWebhook(ctx context.Context, params delivercronwebhook.DeliverCronParams) error
	Logger() logger.Logger
}

// Result holds the output of a cron webhook execution cycle.
type Result struct {
	Triggered int
	Errors    int
}

// cronParser parses standard 5-field cron expressions.
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// Resolve checks for cron webhooks due for execution, creates deliveries, and enqueues jobs.
func Resolve(ctx context.Context, env Env) (*Result, error) {
	webhooks, err := env.ListCronWebhooksDueForExecution(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list cron webhooks due for execution: %w", err)
	}

	result := &Result{}

	for _, wh := range webhooks {
		err = processCronWebhook(ctx, env, wh)
		if err != nil {
			env.Logger().Error("failed to process cron webhook",
				"cron_webhook_id", wh.ID,
				"error", err,
			)
			result.Errors++
			continue
		}
		result.Triggered++
	}

	return result, nil
}

func processCronWebhook(ctx context.Context, env Env, wh db.CronWebhook) error {
	// Parse cron schedule to compute next run time.
	schedule, err := cronParser.Parse(wh.CronSchedule)
	if err != nil {
		return fmt.Errorf("failed to parse cron schedule %q: %w", wh.CronSchedule, err)
	}

	nextRun := schedule.Next(time.Now())

	// Update next_run_at.
	err = env.UpdateCronWebhookNextRunAt(ctx, db.UpdateCronWebhookNextRunAtParams{
		NextRunAt: &nextRun,
		ID:        wh.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update next_run_at: %w", err)
	}

	// Create delivery record.
	delivery, err := env.InsertCronWebhookDelivery(ctx, db.InsertCronWebhookDeliveryParams{
		CronWebhookID: wh.ID,
		Attempt:       1,
	})
	if err != nil {
		return fmt.Errorf("failed to insert cron webhook delivery: %w", err)
	}

	// Enqueue background job.
	err = env.EnqueueDeliverCronWebhook(ctx, delivercronwebhook.DeliverCronParams{
		DeliveryID:    delivery.ID,
		CronWebhookID: wh.ID,
		Attempt:       1,
	})
	if err != nil {
		return fmt.Errorf("failed to enqueue cron webhook delivery: %w", err)
	}

	env.Logger().Info("cron webhook triggered",
		"cron_webhook_id", wh.ID,
		"delivery_id", delivery.ID,
		"next_run_at", nextRun,
	)

	return nil
}
