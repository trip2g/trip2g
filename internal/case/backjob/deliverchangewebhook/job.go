package deliverchangewebhook

import (
	"context"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "deliver_change_webhook"
const QueueID = model.BackgroundDefaultQueue
const Priority = 100

type DeliverChangeWebhookJob struct {
	enqueue jobs.EnqueueFunc
}

type DeliverChangeWebhookEnv interface {
	jobs.Env
	Env
}

func New(env DeliverChangeWebhookEnv) *DeliverChangeWebhookJob {
	return &DeliverChangeWebhookJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (j *DeliverChangeWebhookJob) EnqueueDeliverChangeWebhook(ctx context.Context, params handlenotewebhooks.DeliverChangeWebhookParams) error {
	return j.enqueue(ctx, params)
}
