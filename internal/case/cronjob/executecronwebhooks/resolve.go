package executecronwebhooks

import (
	"context"
	"fmt"
	"time"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/ptr"
	"trip2g/internal/webhookutil"

	"github.com/robfig/cron/v3"
)

type Env interface {
	ListCronWebhooksDueForExecution(ctx context.Context) ([]db.CronWebhook, error)
	UpdateCronWebhookNextRunAt(ctx context.Context, arg db.UpdateCronWebhookNextRunAtParams) error
	InsertCronWebhookDelivery(ctx context.Context, arg db.InsertCronWebhookDeliveryParams) (db.CronWebhookDelivery, error)
	SetCronWebhookDeliveryChain(ctx context.Context, arg db.SetCronWebhookDeliveryChainParams) error
	EnqueueDeliverCronWebhook(ctx context.Context, params delivercronwebhook.DeliverCronParams) error
	Logger() logger.Logger
}

// Result holds the output of a cron webhook execution cycle.
type Result struct {
	Triggered int
	Errors    int
}

//nolint:gochecknoglobals // cronParser is a package-level constant parser.
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

	trace := stampCronDeliveryChain(ctx, env, delivery.ID)

	// Enqueue background job.
	err = env.EnqueueDeliverCronWebhook(ctx, delivercronwebhook.DeliverCronParams{
		DeliveryID:    delivery.ID,
		CronWebhookID: wh.ID,
		Attempt:       1,
		Trace:         trace,
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

// stampCronDeliveryChain marks the delivery as a chain root and returns its
// trace id: a cron run has no cause to point at, and every delivery its writes
// go on to trigger inherits this id. Stamping never blocks the run.
func stampCronDeliveryChain(ctx context.Context, env Env, deliveryID int64) string {
	trace := webhookutil.TraceID(webhookutil.DeliveryKindCron, deliveryID)
	err := env.SetCronWebhookDeliveryChain(ctx, db.SetCronWebhookDeliveryChainParams{
		ID:    deliveryID,
		Trace: ptr.To(trace),
	})
	if err != nil {
		env.Logger().Error("failed to stamp cron delivery chain", "delivery_id", deliveryID, "error", err)
		return ""
	}
	return trace
}
