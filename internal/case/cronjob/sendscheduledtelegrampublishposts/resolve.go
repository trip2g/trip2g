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
	LatestNoteViews() *model.NoteViews

	EnqueueSendTelegramPost(ctx context.Context, params model.SendTelegramPublishPostParams) error
	EnqueueUpdateTelegramPost(ctx context.Context, notePathID int64) error
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
	nvs := env.LatestNoteViews()

	if len(ids) > 1 {
		ids = topologicalsort.ReverseSort(nvs, ids)
		logger.Debug("posts found, sort applied", "count", len(ids))
	} else if len(ids) == 1 {
		logger.Debug("one post found, no sort needed")
	}

	updateIDs := map[int64]struct{}{}

	for _, id := range ids {
		params := model.SendTelegramPublishPostParams{
			NotePathID:        id,
			Instant:           false,
			UpdateLinkedPosts: false,
		}
		sendErr := env.EnqueueSendTelegramPost(ctx, params)

		res.Posts = append(res.Posts, ResultPost{
			NotePathID: id,
			Error:      sendErr,
		})

		if sendErr != nil {
			return nil, fmt.Errorf("failed to EnqueueSendTelegramPost for note_path_id %d: %w", id, sendErr)
		}

		noteView := nvs.GetByPathID(id)

		for inLink := range noteView.InLinks {
			inNote, ok := nvs.Map[inLink]
			if ok && inNote.IsTelegramPublishPost() {
				updateIDs[inNote.PathID] = struct{}{}
			}
		}
	}

	// UpdateTelegramPost has a priority of 0, so these updates will be processed
	// after all SendTelegramPublishPost jobs
	// because the telegram job queue is limited to 1 worker.
	for updateID := range updateIDs {
		err = env.EnqueueUpdateTelegramPost(ctx, updateID)
		if err != nil {
			return nil, fmt.Errorf("failed to EnqueueUpdateTelegramMessage for note_path_id %d: %w", updateID, err)
		}
	}

	return res, nil
}
