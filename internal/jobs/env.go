package jobs

import (
	"context"
	"encoding/json"
	"fmt"
)

type Env interface {
	RegisterJob(id string, handler func(ctx context.Context, m []byte) error)
	EnqueueJob(ctx context.Context, jobID string, data any) error
}

func Register[T any, P any](env Env, jobID string, resolveFunc func(context.Context, T, P) error) {
	_, ok := env.(T)
	if !ok {
		panic("the provided env does not implement the required interface")
	}

	env.RegisterJob(jobID, func(ctx context.Context, m []byte) error {
		var params P

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal %s params: %w", jobID, err)
		}

		err = resolveFunc(ctx, env.(T), params) //nolint:errcheck // backjobs.Env should embeded all env interfaces
		if err != nil {
			return fmt.Errorf("failed to run %s job: %w", jobID, err)
		}

		return nil
	})
}
