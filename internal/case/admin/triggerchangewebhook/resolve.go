package triggerchangewebhook

import (
	"context"
	"fmt"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	internalmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
	"trip2g/internal/webhookutil"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	WebhookByID(ctx context.Context, id int64) (db.ChangeWebhook, error)
	LatestNoteViews() *internalmodel.NoteViews
	InsertWebhookDelivery(ctx context.Context, arg db.InsertWebhookDeliveryParams) (db.ChangeWebhookDelivery, error)
	EnqueueDeliverChangeWebhook(ctx context.Context, params handlenotewebhooks.DeliverChangeWebhookParams) error
}

func Resolve(ctx context.Context, env Env, input model.TriggerChangeWebhookInput) (model.TriggerChangeWebhookOrErrorPayload, error) {
	// Check admin authorization.
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	// Load webhook by ID.
	webhook, err := env.WebhookByID(ctx, input.WebhookID)
	if err != nil {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("Failed to load webhook: %v", err),
		}, nil
	}

	// Check if webhook is enabled.
	if !webhook.Enabled {
		return &model.ErrorPayload{
			Message: "Webhook is not enabled",
		}, nil
	}

	// Get latest note views.
	nvs := env.LatestNoteViews()

	// Parse include/exclude patterns.
	includePatterns, err := webhookutil.ParseJSONStringArray(webhook.IncludePatterns)
	if err != nil {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("Failed to parse include patterns: %v", err),
		}, nil
	}

	excludePatterns, err := webhookutil.ParseJSONStringArray(webhook.ExcludePatterns)
	if err != nil {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("Failed to parse exclude patterns: %v", err),
		}, nil
	}

	// Build ChangeInfo array for each pathId.
	var changeInfos []handlenotewebhooks.ChangeInfo
	var matchedCount int64
	var ignoredCount int64

	for _, pathID := range input.PathIds {
		noteView := nvs.GetByPathID(pathID)
		if noteView == nil {
			// Invalid pathID, skip.
			ignoredCount++
			continue
		}

		// Filter by include/exclude patterns.
		if !webhookutil.MatchesAny(noteView.Path, includePatterns) {
			ignoredCount++
			continue
		}
		if webhookutil.MatchesAny(noteView.Path, excludePatterns) {
			ignoredCount++
			continue
		}

		// Build ChangeInfo.
		info := handlenotewebhooks.ChangeInfo{
			Path:    noteView.Path,
			Event:   "update",
			PathID:  pathID,
			Version: noteView.VersionID,
			Title:   noteView.Title,
		}

		// Include content if enabled.
		if webhook.IncludeContent {
			info.Content = string(noteView.Content)
		}

		changeInfos = append(changeInfos, info)
		matchedCount++
	}

	// If no matches, return error.
	if len(changeInfos) == 0 {
		return &model.ErrorPayload{
			Message: "No paths matched webhook filters",
		}, nil
	}

	// Create delivery record.
	delivery, err := env.InsertWebhookDelivery(ctx, db.InsertWebhookDeliveryParams{
		WebhookID: webhook.ID,
		Attempt:   1,
	})
	if err != nil {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("Failed to create delivery: %v", err),
		}, nil
	}

	// Enqueue job.
	err = env.EnqueueDeliverChangeWebhook(ctx, handlenotewebhooks.DeliverChangeWebhookParams{
		DeliveryID: delivery.ID,
		WebhookID:  webhook.ID,
		Attempt:    1,
		Depth:      0,
		Changes:    changeInfos,
	})
	if err != nil {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("Failed to enqueue job: %v", err),
		}, nil
	}

	return &model.TriggerChangeWebhookPayload{
		MatchedCount: matchedCount,
		IgnoredCount: ignoredCount,
		DeliveryID:   &delivery.ID,
	}, nil
}
