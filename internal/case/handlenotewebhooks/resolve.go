package handlenotewebhooks

import (
	"context"
	"fmt"
	"sort"
	"strings"
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

// webhookStaleMarginSeconds is added on top of a webhook's timeout_seconds when
// computing whether an existing delivery is still legitimately in-flight.
// This small grace period absorbs clock skew and DB write latency.
const webhookStaleMarginSeconds = int64(30)

type Env interface {
	ListEnabledWebhooks(ctx context.Context) ([]db.ChangeWebhook, error)
	InsertWebhookDelivery(ctx context.Context, arg db.InsertWebhookDeliveryParams) (db.ChangeWebhookDelivery, error)
	InsertWebhookDeliveryIfClear(ctx context.Context, arg db.InsertWebhookDeliveryIfClearParams) (db.ChangeWebhookDelivery, error)
	InsertWebhookDeliveryIfNoPending(ctx context.Context, webhookID int64) (db.ChangeWebhookDelivery, error)
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

// attachGateSatisfied reports whether the webhook's attach_notes preconditions
// hold against the current note set. A plain glob requires >=1 matching note;
// a "!glob" requires 0 matching notes. Empty attach is always satisfied.
func attachGateSatisfied(attach []string, nvs *model.NoteViews) bool {
	for _, pat := range attach {
		if strings.HasPrefix(pat, "!") {
			if anyNoteMatches(strings.TrimPrefix(pat, "!"), nvs) {
				return false // required-absent glob matched something
			}
			continue
		}
		if !anyNoteMatches(pat, nvs) {
			return false // required-present glob matched nothing
		}
	}
	return true
}

func anyNoteMatches(glob string, nvs *model.NoteViews) bool {
	if nvs == nil {
		return false
	}
	for path := range nvs.PathMap {
		if webhookutil.MatchesAny(path, []string{glob}) {
			return true
		}
	}
	return false
}

// AttachGateSatisfiedForTest exposes attachGateSatisfied to the test package.
func AttachGateSatisfiedForTest(attach []string, nvs *model.NoteViews) bool {
	return attachGateSatisfied(attach, nvs)
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

		// attach_notes presence gate: skip if the role's required context is
		// absent (plain glob) or a forbidden note is present ("!glob").
		var attach []string
		if wh.AttachNotes != "" {
			var attachErr error
			attach, attachErr = webhookutil.ParseJSONStringArray(wh.AttachNotes)
			if attachErr != nil {
				env.Logger().Error("failed to parse attach_notes", "webhook_id", wh.ID, "error", attachErr)
				continue
			}
		}
		if !attachGateSatisfied(attach, nvs) {
			continue
		}

		// Sort by path for deterministic ordering.
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].Path < matched[j].Path
		})

		// Create delivery record, respecting the webhook's concurrency_mode.
		// For skip mode the stale window is derived from the webhook's own timeout_seconds
		// (+ a small margin), so a long-running delivery is not treated as stale prematurely.
		staleWindow := fmt.Sprintf("-%d seconds", wh.TimeoutSeconds+webhookStaleMarginSeconds)

		var delivery db.ChangeWebhookDelivery
		var insertErr error
		switch wh.ConcurrencyMode {
		case "skip":
			delivery, insertErr = env.InsertWebhookDeliveryIfClear(ctx, db.InsertWebhookDeliveryIfClearParams{
				WebhookID:   wh.ID,
				StaleWindow: staleWindow,
			})
		case "queue_one":
			delivery, insertErr = env.InsertWebhookDeliveryIfNoPending(ctx, wh.ID)
		default: // allow_overlap
			delivery, insertErr = env.InsertWebhookDelivery(ctx, db.InsertWebhookDeliveryParams{
				WebhookID: wh.ID,
				Attempt:   1,
			})
		}
		if db.IsNoFound(insertErr) {
			// skip/queue_one: nothing inserted because an in-flight/pending exists.
			continue
		}
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
