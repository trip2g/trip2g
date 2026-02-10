package handlenotewebhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/bmatcuk/doublestar/v4"
)

// NoteChange describes a single note change event.
type NoteChange struct {
	PathID int64
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
		includePatterns, err := parseJSONStringArray(wh.IncludePatterns)
		if err != nil {
			env.Logger().Error("failed to parse include_patterns", "webhook_id", wh.ID, "error", err)
			continue
		}

		excludePatterns, err := parseJSONStringArray(wh.ExcludePatterns)
		if err != nil {
			env.Logger().Error("failed to parse exclude_patterns", "webhook_id", wh.ID, "error", err)
			continue
		}

		// Filter changes by event type and glob patterns.
		var matched []ChangeInfo

		for _, ch := range changes {
			// Event type filtering.
			switch ch.Event {
			case "create":
				if !wh.OnCreate {
					continue
				}
			case "update":
				if !wh.OnUpdate {
					continue
				}
			case "remove":
				if !wh.OnRemove {
					continue
				}
			}

			// Get note view for path info.
			noteView := nvs.GetByPathID(ch.PathID)
			if noteView == nil {
				// Note not found in latest views (might have been deleted).
				continue
			}

			path := noteView.Path

			// Glob matching: include patterns.
			if !matchesAny(path, includePatterns) {
				continue
			}

			// Glob matching: exclude patterns.
			if matchesAny(path, excludePatterns) {
				continue
			}

			info := ChangeInfo{
				Path:    path,
				Event:   ch.Event,
				PathID:  noteView.PathID,
				Version: noteView.VersionID,
				Title:   noteView.Title,
			}

			// Include content if enabled and not a remove event.
			if wh.IncludeContent && ch.Event != "remove" {
				info.Content = string(noteView.Content)
			}

			matched = append(matched, info)
		}

		if len(matched) == 0 {
			continue
		}

		// Sort by path for deterministic ordering.
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].Path < matched[j].Path
		})

		// Create delivery record.
		delivery, err := env.InsertWebhookDelivery(ctx, db.InsertWebhookDeliveryParams{
			WebhookID: wh.ID,
			Attempt:   1,
		})
		if err != nil {
			env.Logger().Error("failed to insert webhook delivery", "webhook_id", wh.ID, "error", err)
			continue
		}

		// Enqueue background job.
		err = env.EnqueueDeliverChangeWebhook(ctx, DeliverChangeWebhookParams{
			DeliveryID: delivery.ID,
			WebhookID:  wh.ID,
			Attempt:    1,
			Depth:      depth,
			Changes:    matched,
		})
		if err != nil {
			env.Logger().Error("failed to enqueue webhook delivery", "webhook_id", wh.ID, "delivery_id", delivery.ID, "error", err)
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

// matchesAny checks if path matches any of the glob patterns.
func matchesAny(path string, patterns []string) bool {
	for _, p := range patterns {
		matched, err := doublestar.Match(p, path)
		if err != nil {
			// Invalid pattern — skip it.
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// parseJSONStringArray parses a JSON string array like '["blog/**","docs/*"]'.
func parseJSONStringArray(raw string) ([]string, error) {
	var result []string

	err := json.Unmarshal([]byte(raw), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON string array: %w", err)
	}

	return result, nil
}
