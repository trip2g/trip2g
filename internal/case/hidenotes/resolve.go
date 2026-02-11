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
)

type Env interface {
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	LatestNoteViews() *internalmodel.NoteViews
	Logger() logger.Logger
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

	result := &model.HideNotesPayload{
		Success: true,
	}

	// Trigger remove webhooks after successful hide.
	triggerWebhooks(ctx, env, webhookChanges)

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
