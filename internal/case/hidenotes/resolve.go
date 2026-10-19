package hidenotes

import (
	"context"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/case/updatenotes"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	internalmodel "trip2g/internal/model"
	"trip2g/internal/notebus"
)

type Env interface {
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	LatestNoteContentHash(ctx context.Context, path string) (string, error)
	LatestNoteViews() *internalmodel.NoteViews
	PrepareLatestNotes(ctx context.Context, partial bool) (*internalmodel.NoteViews, error)
	Logger() logger.Logger
	PublishNoteChanges(batch notebus.Batch)
	HandleNoteWebhooks(ctx context.Context, changes []handlenotewebhooks.NoteChange, depth int) error
}

type Input = model.HideNotesInput
type Payload = model.HideNotesOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// hideNotes is a bulk-hide infra mutation with no per-path scoping — scoped
	// tokens are denied outright (updateNotes' hide change is the scoped path).
	if err := appreq.RequireUnscoped(ctx); err != nil {
		return &model.ErrorPayload{Message: err.Error()}, nil //nolint:nilerr // authorization denial returned as payload, not as error.
	}

	if input.ApiKey.CreatedBy == 0 {
		return &model.ErrorPayload{Message: updatenotes.NoHideActorMessage}, nil
	}

	targets := make([]model.HideNoteInput, 0, len(input.Paths)+len(input.Notes))
	for _, path := range input.Paths {
		targets = append(targets, model.HideNoteInput{Path: path})
	}
	targets = append(targets, input.Notes...)

	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if _, dup := seen[target.Path]; dup {
			return &model.ErrorPayload{Message: fmt.Sprintf("path %s is listed more than once", target.Path)}, nil
		}
		seen[target.Path] = struct{}{}
	}

	// Collect note info before hiding (notes may be removed from NoteViews after reload).
	nvs := env.LatestNoteViews()
	var webhookChanges []handlenotewebhooks.NoteChange
	var hashMismatches []model.NoteHashMismatch

	for _, target := range targets {
		// The request runs in one write transaction on the single write
		// connection, so no other write lands between this read and the hide.
		if target.ExpectedHash != nil {
			actualHash, err := env.LatestNoteContentHash(ctx, target.Path)
			if err != nil {
				return nil, err
			}

			if actualHash != *target.ExpectedHash {
				hashMismatches = append(hashMismatches, model.NoteHashMismatch{Path: target.Path, ActualHash: actualHash})
				continue
			}

			// "" matched: the note is absent or already hidden, so there is
			// nothing to hide and its original hidden_by stays.
			if actualHash == "" {
				continue
			}
		}

		params := db.HideNotePathParams{
			HiddenBy: &input.ApiKey.CreatedBy,
			Value:    target.Path,
		}

		err := env.HideNotePath(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to hide note path %s: %w", target.Path, err)
		}

		change := handlenotewebhooks.NoteChange{
			Path:  target.Path,
			Event: "remove",
		}
		if nv := nvs.GetByPath(target.Path); nv != nil {
			change.PathID = nv.PathID
		}
		webhookChanges = append(webhookChanges, change)
	}

	// Reload NoteViews so hidden paths stop resolving on the public site.
	// rendernotepage serves from the in-memory cache; without a reload the
	// hidden note keeps returning 200 until the next full reload (asset-URL
	// expiry or restart). The RawNotes source query filters hidden_by IS NULL.
	if len(webhookChanges) > 0 {
		if _, err := env.PrepareLatestNotes(ctx, false); err != nil {
			return nil, fmt.Errorf("failed to reload notes after hide: %w", err)
		}
	}

	result := &model.HideNotesPayload{
		Success:        true,
		HashMismatches: hashMismatches,
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

	webhookErr := env.HandleNoteWebhooks(ctx, changes, req.WebhookDepth)
	if webhookErr != nil {
		env.Logger().Error("failed to handle note webhooks for hide", "error", webhookErr)
	}
}
