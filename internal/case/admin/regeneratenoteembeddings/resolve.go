package regeneratenoteembeddings

import (
	"context"
	"fmt"

	cronregen "trip2g/internal/case/cronjob/regeneratenoteembeddings"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	cronregen.Env
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.RegenerateNoteEmbeddingsInput
type Payload = model.RegenerateNoteEmbeddingsOrErrorPayload

// Resolve runs the same pass as the regenerate_note_embeddings cronjob, on
// demand. With force it enqueues every note instead of only those whose content
// hash moved — the manual lever for a change the hash does not cover, such as
// new chunking. Enqueueing is all it does: the jobs drain through the global
// queue at its own concurrency, so one instance at a time can be repaired
// without every instance sharing an embedding model deciding to do it at once.
func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	run := cronregen.Resolve
	if input.Force != nil && *input.Force {
		run = cronregen.ResolveForced
	}

	result, err := run(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate note embeddings: %w", err)
	}
	if len(result.Errors) > 0 {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("enqueued %d of %d notes; %d failed: %v",
				result.EnqueuedCount, result.TotalNotes, len(result.Errors), result.Errors[0]),
		}, nil
	}

	return &model.RegenerateNoteEmbeddingsPayload{
		TotalNotes: int32(result.TotalNotes),
		Enqueued:   int32(result.EnqueuedCount),
		UpToDate:   int32(result.UpToDateCount),
	}, nil
}
