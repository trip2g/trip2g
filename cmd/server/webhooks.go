package main

import (
	"context"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/case/handletgpublishviews"
	"trip2g/internal/case/updatesubgraphs"
	"trip2g/internal/notebus"

	"github.com/valyala/fasthttp"
)

func (a *app) HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error {
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
		nvs := a.LatestNoteViews()
		for _, pathID := range changedPathIDs {
			noteView := nvs.GetByPathID(pathID)
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
