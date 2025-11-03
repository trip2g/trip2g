package sendscheduledtelegrampublishposts

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger
	ListSheduledTelegarmPublishNoteIDs(ctx context.Context) ([]int64, error)
	QueueSendPublishPost(ctx context.Context, notePathID int64, instant bool) error
}

type ResultPost struct {
	NotePathID int64 `json:"note_path_id"`
	Error      error `json:"error,omitempty"`
}

type Result struct {
	Posts []ResultPost `json:"posts"`
}

func Resolve(ctx context.Context, env Env) (any, error) {
	logger := logger.WithPrefix(env.Logger(), "sendscheduledtelegrampublishposts:")

	ids, err := env.ListSheduledTelegarmPublishNoteIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListSheduledTelegarmPublishNoteIDs: %w", err)
	}

	res := Result{}

	logger.Debug("posts found", "count", len(ids))

	for _, id := range ids {
		sendErr := env.QueueSendPublishPost(ctx, id, false)

		res.Posts = append(res.Posts, ResultPost{
			NotePathID: id,
			Error:      sendErr,
		})

		if sendErr != nil {
			logger.Error("failed to QueueSendPublishPost", "note_path_id", id, "error", sendErr)
		}
	}

	return res, nil
}
