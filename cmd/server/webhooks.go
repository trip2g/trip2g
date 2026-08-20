package main

import (
	"context"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/case/handletgpublishviews"
	"trip2g/internal/case/materializenotefrontmatters"
	"trip2g/internal/case/updatesubgraphs"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/notebus"
	"trip2g/internal/webhookutil"

	"github.com/valyala/fasthttp"
)

//nolint:gocognit // This coordinates independent post-save side effects.
func (a *app) HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error {
	nvs := a.LatestNoteViews()
	changed := make([]*model.NoteView, 0, len(changedPathIDs))
	for _, pathID := range changedPathIDs {
		if note := nvs.GetByPathID(pathID); note != nil {
			changed = append(changed, note)
		}
	}
	if err := materializenotefrontmatters.Resolve(ctx, a, changed); err != nil {
		return fmt.Errorf("failed to materialize note frontmatters: %w", err)
	}

	err := updatesubgraphs.Resolve(ctx, a)
	if err != nil {
		return fmt.Errorf("failed to update subgraphs: %w", err)
	}

	err = handletgpublishviews.Resolve(ctx, a, changedPathIDs)
	if err != nil {
		return fmt.Errorf("failed to handle Telegram publish views: %w", err)
	}

	// Enqueue embedding generation for changed notes (if vector search enabled)
	if a.config.Features.VectorSearch.Enabled {
		latestNotes := a.LatestNoteViews()
		for _, pathID := range changedPathIDs {
			noteView := latestNotes.GetByPathID(pathID)
			if noteView != nil {
				enqueueErr := a.GenerateNoteVersionEmbeddingJob.Enqueue(ctx, noteView.VersionID)
				if enqueueErr != nil {
					a.log.Error("failed to enqueue embedding generation", "version_id", noteView.VersionID, "error", enqueueErr)
				}
			}
		}
	}

	// Trigger change webhook deliveries for changed notes.
	// Check if the current request has skip_webhooks enabled.
	req, reqErr := appreq.FromCtx(ctx)
	skipWebhooks := reqErr == nil && req.SkipWebhooks
	webhookDepth := 0
	if reqErr == nil {
		webhookDepth = req.WebhookDepth
	}

	if skipWebhooks {
		return nil
	}

	webhookChanges := make([]handlenotewebhooks.NoteChange, 0, len(changedPathIDs))
	for _, pathID := range changedPathIDs {
		notePath, npErr := a.NotePathByID(ctx, pathID)
		if npErr != nil {
			a.log.Error("failed to get note path for webhook", "path_id", pathID, "error", npErr)
			continue
		}

		event := "update"
		if notePath.VersionCount == 1 {
			event = "create"
		}

		webhookChanges = append(webhookChanges, handlenotewebhooks.NoteChange{
			PathID: pathID,
			Event:  event,
		})
	}

	// Publish to note change bus for SSE subscribers.
	var busBatch notebus.Batch
	for _, wc := range webhookChanges {
		path := ""
		nv := a.LatestNoteViews().GetByPathID(wc.PathID)
		if nv != nil {
			path = nv.Path
		}
		busBatch.Changes = append(busBatch.Changes, notebus.Change{
			PathID: wc.PathID,
			Path:   path,
			Event:  wc.Event,
		})
	}
	if len(busBatch.Changes) > 0 {
		a.PublishNoteChanges(busBatch)
	}

	if len(webhookChanges) > 0 {
		webhookErr := handlenotewebhooks.Resolve(ctx, a, webhookChanges, webhookDepth)
		if webhookErr != nil {
			a.log.Error("failed to handle note webhooks", "error", webhookErr)
		}
	}

	return nil
}

// DeliveryTrace returns the trace id of an existing delivery, dispatching on
// its kind (delivery ids are only unique within their own table). A delivery
// that predates the chain columns, or one that vanished with its retention
// window, yields an empty trace: the caller then starts a fresh chain rather
// than failing the delivery it is about to create.
func (a *app) DeliveryTrace(ctx context.Context, kind string, deliveryID int64) (string, error) {
	var trace *string
	switch kind {
	case webhookutil.DeliveryKindChange:
		row, err := a.WebhookDeliveryTraceByID(ctx, deliveryID)
		if db.IsNoFound(err) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		trace = row.Trace
	case webhookutil.DeliveryKindCron:
		row, err := a.CronWebhookDeliveryTraceByID(ctx, deliveryID)
		if db.IsNoFound(err) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		trace = row.Trace
	default:
		return "", nil
	}
	if trace == nil {
		return "", nil
	}
	return *trace, nil
}

func (a *app) SubscribeNoteChanges(include, exclude []string) *notebus.Subscriber {
	return a.noteBus.Subscribe(include, exclude, 64)
}

func (a *app) UnsubscribeNoteChanges(sub *notebus.Subscriber) {
	a.noteBus.Unsubscribe(sub)
}

func (a *app) PublishNoteChanges(batch notebus.Batch) {
	a.noteBus.Publish(batch)
}

// WebhookHTTPClient returns the shared HTTP client for webhook deliveries.
func (a *app) WebhookHTTPClient() *fasthttp.Client {
	return a.webhookHTTPClient
}

// EnqueueDeliverChangeWebhook enqueues a change webhook delivery job.
func (a *app) EnqueueDeliverChangeWebhook(ctx context.Context, params handlenotewebhooks.DeliverChangeWebhookParams) error {
	return a.DeliverChangeWebhookJob.EnqueueDeliverChangeWebhook(ctx, params)
}

// EnqueueDeliverCronWebhook enqueues a cron webhook delivery job.
func (a *app) EnqueueDeliverCronWebhook(ctx context.Context, params delivercronwebhook.DeliverCronParams) error {
	return a.DeliverCronWebhookJob.EnqueueDeliverCronWebhook(ctx, params)
}

// ExpireStaleWebhookDeliveries marks orphaned 'running'/'pending' change webhook
// deliveries as 'failed' once their per-webhook liveness window (timeout_seconds
// + 30-second margin) has lapsed.
func (a *app) ExpireStaleWebhookDeliveries(ctx context.Context) error {
	return a.WriteQueries.ExpireStaleWebhookDeliveries(ctx)
}

// ExpireStaleCronWebhookDeliveries marks orphaned 'running'/'pending' cron webhook
// deliveries as 'failed' once their per-webhook liveness window (timeout_seconds
// + 30-second margin) has lapsed.
func (a *app) ExpireStaleCronWebhookDeliveries(ctx context.Context) error {
	return a.WriteQueries.ExpireStaleCronWebhookDeliveries(ctx)
}
