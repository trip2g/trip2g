package hidenotes

import (
	"context"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	internalmodel "trip2g/internal/model"
	"trip2g/internal/notebus"
)

type Env interface {
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	LatestNoteViews() *internalmodel.NoteViews
	PrepareLatestNotes(ctx context.Context, partial bool) (*internalmodel.NoteViews, error)
	Logger() logger.Logger
	PublishNoteChanges(batch notebus.Batch)
}

type Input = model.HideNotesInput
type Payload = model.HideNotesOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Collect note info before hiding (notes may be removed from NoteViews after reload).
	nvs := env.LatestNoteViews()
	var webhookChanges []handlenotewebhooks.NoteChange
	for _, path := range input.Paths {
		nv := nvs.GetByPath(path)
		change := handlenotewebhooks.NoteChange{
			Path:  path,
			Event: "remove",
		}
		if nv != nil {
			change.PathID = nv.PathID
		}
		webhookChanges = append(webhookChanges, change)
	}

	// Perform the hide operation.
	for _, path := range input.Paths {
		params := db.HideNotePathParams{
			HiddenBy: &input.ApiKey.CreatedBy,
			Value:    path,
		}

		err := env.HideNotePath(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to hide note path %s: %w", path, err)
		}
	}

	// Reload NoteViews so hidden paths stop resolving on the public site.
	// rendernotepage serves from the in-memory cache; without a reload the
	// hidden note keeps returning 200 until the next full reload (asset-URL
	// expiry or restart). The RawNotes source query filters hidden_by IS NULL.
	if len(input.Paths) > 0 {
		if _, err := env.PrepareLatestNotes(ctx, false); err != nil {
			return nil, fmt.Errorf("failed to reload notes after hide: %w", err)
		}
	}

	result := &model.HideNotesPayload{
		Success: true,
	}

	// Trigger remove webhooks after successful hide.
	triggerWebhooks(ctx, env, webhookChanges)

	// Publish to note change bus for SSE subscribers.
	var busBatch notebus.Batch
	for _, wc := range webhookChanges {
		busBatch.Changes = append(busBatch.Changes, notebus.Change{
			PathID: wc.PathID,
			Path:   wc.Path,
			Event:  "remove",
		})
	}
	if len(busBatch.Changes) > 0 {
		env.PublishNoteChanges(busBatch)
	}

	return result, nil
}

// triggerWebhooks sends webhook notifications for removed notes.
func triggerWebhooks(ctx context.Context, env Env, changes []handlenotewebhooks.NoteChange) {
	if len(changes) == 0 {
		return
	}

	req, reqErr := appreq.FromCtx(ctx)
	if reqErr != nil {
		return
	}
	if req.SkipWebhooks {
		return
	}

	webhookEnv, ok := req.Env.(handlenotewebhooks.Env)
	if !ok {
		env.Logger().Error("failed to cast env to handlenotewebhooks.Env for hide webhooks")
		return
	}

	webhookErr := handlenotewebhooks.Resolve(ctx, webhookEnv, changes, req.WebhookDepth)
	if webhookErr != nil {
		env.Logger().Error("failed to handle note webhooks for hide", "error", webhookErr)
	}
}
