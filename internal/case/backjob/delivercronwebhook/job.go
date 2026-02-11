package delivercronwebhook

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "deliver_cron_webhook"
const QueueID = model.BackgroundDefaultQueue
const Priority = 100

// DeliverCronParams holds the job parameters.
type DeliverCronParams struct {
	DeliveryID    int64  `json:"delivery_id"`
	CronWebhookID int64  `json:"cron_webhook_id"`
	Attempt       int    `json:"attempt"`
	PreviousError string `json:"previous_error,omitempty"`
}

type DeliverCronWebhookJob struct {
	enqueue jobs.EnqueueFunc
}

type DeliverCronWebhookEnv interface {
	jobs.Env
	Env
}

func New(env DeliverCronWebhookEnv) *DeliverCronWebhookJob {
	return &DeliverCronWebhookJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (j *DeliverCronWebhookJob) EnqueueDeliverCronWebhook(ctx context.Context, params DeliverCronParams) error {
	return j.enqueue(ctx, params)
}
