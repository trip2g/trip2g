package stopbackgroundqueue

import (
	"context"
	"fmt"

	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	StopBackgroundQueue(ctx context.Context, name string) error
	GetBackgroundQueue(ctx context.Context, name string) (*appmodel.BackgroundQueue, error)
	ListBackgroundQueues(ctx context.Context) []appmodel.BackgroundQueue
}

type Input = model.StopBackgroundQueueInput
type Payload = model.StopBackgroundQueueOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}
	_ = token // token validated, unused for now but available if needed

	err = env.StopBackgroundQueue(ctx, input.ID)
	if err != nil {
		//nolint:nilerr // ErrorPayload is the GraphQL way to return user-facing errors
		return &model.ErrorPayload{Message: err.Error()}, nil
	}

	var queues []appmodel.BackgroundQueue

	// If "*" was used, return all queues
	if input.ID == "*" {
		queues = env.ListBackgroundQueues(ctx)
	} else {
		// Single queue
		var q *appmodel.BackgroundQueue
		q, err = env.GetBackgroundQueue(ctx, input.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get queue after stop: %w", err)
		}
		queues = []appmodel.BackgroundQueue{*q}
	}

	payload := model.StopBackgroundQueuePayload{
		Queues: queues,
	}

	return &payload, nil
}
