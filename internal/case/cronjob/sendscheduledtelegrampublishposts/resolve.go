package sendscheduledtelegrampublishposts

import (
	"context"
	"fmt"
)

type Env interface {
	ListSheduledTelegarmPublishNoteIDs(ctx context.Context) ([]int64, error)
	SendTelegramPublishPost(ctx context.Context, notePathID int64) error
}

func Resolve(ctx context.Context, env Env) (any, error) {
	ids, err := env.ListSheduledTelegarmPublishNoteIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListSheduledTelegarmPublishNoteIDs: %w", err)
	}

	for _, id := range ids {
		err = env.SendTelegramPublishPost(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to SendTelegramPublishPost: %w", err)
		}
	}

	return nil, nil
}
