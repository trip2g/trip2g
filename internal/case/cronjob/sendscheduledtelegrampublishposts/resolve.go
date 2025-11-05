package sendscheduledtelegrampublishposts

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/topologicalsort"
)

type Env interface {
	Logger() logger.Logger
	ListSheduledTelegarmPublishNoteIDs(ctx context.Context) ([]int64, error)
	EnqueueSendPublishPost(ctx context.Context, notePathID int64, instant bool) error
	LatestNoteViews() *model.NoteViews
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

	if len(ids) > 1 {
		ids = topologicalsort.ReverseSort(env.LatestNoteViews(), ids)
		logger.Debug("posts found, sort applied", "count", len(ids))
	} else if len(ids) == 1 {
		logger.Debug("one post found, no sort needed")
	}

	for _, id := range ids {
		sendErr := env.EnqueueSendPublishPost(ctx, id, false)

		res.Posts = append(res.Posts, ResultPost{
			NotePathID: id,
			Error:      sendErr,
		})

		if sendErr != nil {
			logger.Error("failed to EnqueueSendPublishPost", "note_path_id", id, "error", sendErr)
		}
	}

	return res, nil
}
