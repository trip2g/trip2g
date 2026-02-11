package handlenotewebhooks

import (
	"context"
	"fmt"
	"sort"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/webhookutil"
)

const eventRemove = "remove"

// NoteChange describes a single note change event.
type NoteChange struct {
	PathID int64
	Path   string // Used for "remove" events when note is no longer in NoteViews.
	Event  string // "create", "update", "remove"
}

// ChangeInfo is the per-note data passed to the delivery job.
type ChangeInfo struct {
	Path    string `json:"path"`
	Event   string `json:"event"`
	PathID  int64  `json:"path_id"`
	Version int64  `json:"version"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
}

// DeliverChangeWebhookParams is the job parameter for deliverchangewebhook.
type DeliverChangeWebhookParams struct {
	DeliveryID    int64        `json:"delivery_id"`
	WebhookID     int64        `json:"webhook_id"`
	Attempt       int          `json:"attempt"`
	Depth         int          `json:"depth"`
	Changes       []ChangeInfo `json:"changes"`
	PreviousError string       `json:"previous_error,omitempty"`
}

type Env interface {
	ListEnabledWebhooks(ctx context.Context) ([]db.ChangeWebhook, error)
	InsertWebhookDelivery(ctx context.Context, arg db.InsertWebhookDeliveryParams) (db.ChangeWebhookDelivery, error)
	LatestNoteViews() *model.NoteViews
	EnqueueDeliverChangeWebhook(ctx context.Context, params DeliverChangeWebhookParams) error
	Logger() logger.Logger
}

// matchChange checks if a single change matches the webhook's filters and returns a ChangeInfo if it does.
func matchChange(ch NoteChange, wh db.ChangeWebhook, nvs *model.NoteViews, includePatterns, excludePatterns []string) *ChangeInfo {
	// Event type filtering.
	switch ch.Event {
	case "create":
		if !wh.OnCreate {
			return nil
		}
	case "update":
		if !wh.OnUpdate {
			return nil
		}
	case eventRemove:
		if !wh.OnRemove {
			return nil
		}
	}

	// Get note view for path info.
	noteView := nvs.GetByPathID(ch.PathID)
	if noteView == nil {
		if ch.Event != eventRemove || ch.Path == "" {
			return nil
		}
	}

	// Determine path.
	var path string
	if noteView != nil {
		path = noteView.Path
	} else {
		path = ch.Path
	}

	// Glob matching.
	if !webhookutil.MatchesAny(path, includePatterns) {
		return nil
	}
	if webhookutil.MatchesAny(path, excludePatterns) {
		return nil
	}

	info := ChangeInfo{
		Path:   path,
		Event:  ch.Event,
		PathID: ch.PathID,
	}

	if noteView != nil {
		info.Version = noteView.VersionID
		info.Title = noteView.Title
	}

	// Include content if enabled and not a remove event.
	if wh.IncludeContent && ch.Event != eventRemove && noteView != nil {
		info.Content = string(noteView.Content)
	}

	return &info
}

// Resolve processes changed notes against enabled webhooks.
// It filters by depth, event type, and glob patterns, then creates
// delivery records and enqueues background jobs for matching webhooks.
func Resolve(ctx context.Context, env Env, changes []NoteChange, depth int) error {
	if len(changes) == 0 {
		return nil
	}

	webhooks, err := env.ListEnabledWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list enabled webhooks: %w", err)
	}

	if len(webhooks) == 0 {
		return nil
	}

	nvs := env.LatestNoteViews()

	for _, wh := range webhooks {
		// Depth check: skip if current depth is too deep for this webhook.
		if int64(depth) >= wh.MaxDepth {
			continue
		}

		// Parse include/exclude patterns from JSON.
		includePatterns, parseErr := webhookutil.ParseJSONStringArray(wh.IncludePatterns)
		if parseErr != nil {
			env.Logger().Error("failed to parse include_patterns", "webhook_id", wh.ID, "error", parseErr)
			continue
		}

		excludePatterns, parseErr := webhookutil.ParseJSONStringArray(wh.ExcludePatterns)
		if parseErr != nil {
			env.Logger().Error("failed to parse exclude_patterns", "webhook_id", wh.ID, "error", parseErr)
			continue
		}

		// Filter changes by event type and glob patterns.
		var matched []ChangeInfo

		for _, ch := range changes {
			info := matchChange(ch, wh, nvs, includePatterns, excludePatterns)
			if info != nil {
				matched = append(matched, *info)
			}
		}

		if len(matched) == 0 {
			continue
		}

		// Sort by path for deterministic ordering.
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].Path < matched[j].Path
		})

		// Create delivery record.
		delivery, insertErr := env.InsertWebhookDelivery(ctx, db.InsertWebhookDeliveryParams{
			WebhookID: wh.ID,
			Attempt:   1,
		})
		if insertErr != nil {
			env.Logger().Error("failed to insert webhook delivery", "webhook_id", wh.ID, "error", insertErr)
			continue
		}

		// Enqueue background job.
		enqueueErr := env.EnqueueDeliverChangeWebhook(ctx, DeliverChangeWebhookParams{
			DeliveryID: delivery.ID,
			WebhookID:  wh.ID,
			Attempt:    1,
			Depth:      depth,
			Changes:    matched,
		})
		if enqueueErr != nil {
			env.Logger().Error("failed to enqueue webhook delivery", "webhook_id", wh.ID, "delivery_id", delivery.ID, "error", enqueueErr)
			continue
		}

		env.Logger().Info("webhook delivery enqueued",
			"webhook_id", wh.ID,
			"delivery_id", delivery.ID,
			"matched_count", len(matched),
		)
	}

	return nil
}
