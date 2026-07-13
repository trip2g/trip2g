package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"trip2g/internal/model"
)

type Env interface {
	RegisterJob(qID model.BackgroundQueueID, id string, handler func(ctx context.Context, m []byte) error)
	EnqueueJob(ctx context.Context, job model.BackgroundTask) error
}

type EnqueueFunc func(ctx context.Context, data any) error

func Register[P any](
	env Env,
	qID model.BackgroundQueueID,
	jobID string,
	priority int,
	handler func(context.Context, P) error,
) EnqueueFunc {
	env.RegisterJob(qID, jobID, func(ctx context.Context, m []byte) error {
		var params P

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal %s params: %w", jobID, err)
		}

		err = handler(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to run %s job: %w", jobID, err)
		}

		return nil
	})

	return func(ctx context.Context, data any) error {
		return env.EnqueueJob(ctx, model.BackgroundTask{
			ID:       jobID,
			Queue:    qID,
			Data:     data,
			Priority: priority,
		})
	}
}
