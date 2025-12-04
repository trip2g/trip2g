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
	LatestNoteViews() *model.NoteViews

	// Bot publishing
	ListSheduledTelegarmPublishNoteIDs(ctx context.Context) ([]int64, error)
	EnqueueSendTelegramPost(ctx context.Context, params model.SendTelegramPublishPostParams) error
	EnqueueUpdateTelegramPost(ctx context.Context, notePathID int64) error

	// Account publishing
	ListSheduledTelegarmAccountPublishNoteIDs(ctx context.Context) ([]int64, error)
	EnqueueSendTelegramAccountPost(ctx context.Context, params model.SendTelegramPublishPostParams) error
	EnqueueUpdateTelegramAccountPost(ctx context.Context, notePathID int64) error
}

type ResultPost struct {
	NotePathID int64  `json:"note_path_id"`
	Type       string `json:"type"` // "bot" or "account"
	Error      error  `json:"error,omitempty"`
}

type Result struct {
	BotPosts     []ResultPost `json:"bot_posts"`
	AccountPosts []ResultPost `json:"account_posts"`
}

func Resolve(ctx context.Context, env Env) (any, error) {
	res := Result{}

	botPosts, err := enqueueBotJobs(ctx, env)
	if err != nil {
		return nil, err
	}
	res.BotPosts = botPosts

	accountPosts, err := enqueueAccountJobs(ctx, env)
	if err != nil {
		return nil, err
	}
	res.AccountPosts = accountPosts

	return res, nil
}

func enqueueBotJobs(ctx context.Context, env Env) ([]ResultPost, error) {
	log := logger.WithPrefix(env.Logger(), "sendscheduledtelegrampublishposts:bot:")

	ids, err := env.ListSheduledTelegarmPublishNoteIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListSheduledTelegarmPublishNoteIDs: %w", err)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	nvs := env.LatestNoteViews()

	if len(ids) > 1 {
		ids = topologicalsort.ReverseSort(nvs, ids)
		log.Debug("posts found, sort applied", "count", len(ids))
	} else {
		log.Debug("one post found, no sort needed")
	}

	var posts []ResultPost
	updateIDs := map[int64]struct{}{}

	for _, id := range ids {
		params := model.SendTelegramPublishPostParams{
			NotePathID:        id,
			Instant:           false,
			UpdateLinkedPosts: false,
		}

		sendErr := env.EnqueueSendTelegramPost(ctx, params)
		if sendErr != nil {
			posts = append(posts, ResultPost{
				NotePathID: id,
				Type:       "bot",
				Error:      sendErr,
			})
			return posts, fmt.Errorf("failed to EnqueueSendTelegramPost for note_path_id %d: %w", id, sendErr)
		}

		posts = append(posts, ResultPost{
			NotePathID: id,
			Type:       "bot",
		})

		noteView := nvs.GetByPathID(id)
		if noteView != nil {
			for inLink := range noteView.InLinks {
				inNote, ok := nvs.Map[inLink]
				if ok && inNote.IsTelegramPublishPost() {
					updateIDs[inNote.PathID] = struct{}{}
				}
			}
		}
	}

	for updateID := range updateIDs {
		err = env.EnqueueUpdateTelegramPost(ctx, updateID)
		if err != nil {
			return posts, fmt.Errorf("failed to EnqueueUpdateTelegramPost for note_path_id %d: %w", updateID, err)
		}
	}

	return posts, nil
}

func enqueueAccountJobs(ctx context.Context, env Env) ([]ResultPost, error) {
	log := logger.WithPrefix(env.Logger(), "sendscheduledtelegrampublishposts:account:")

	ids, err := env.ListSheduledTelegarmAccountPublishNoteIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListSheduledTelegarmAccountPublishNoteIDs: %w", err)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	nvs := env.LatestNoteViews()

	if len(ids) > 1 {
		ids = topologicalsort.ReverseSort(nvs, ids)
		log.Debug("posts found, sort applied", "count", len(ids))
	} else {
		log.Debug("one post found, no sort needed")
	}

	var posts []ResultPost
	updateIDs := map[int64]struct{}{}

	for _, id := range ids {
		params := model.SendTelegramPublishPostParams{
			NotePathID:        id,
			Instant:           false,
			UpdateLinkedPosts: false,
		}

		sendErr := env.EnqueueSendTelegramAccountPost(ctx, params)
		if sendErr != nil {
			posts = append(posts, ResultPost{
				NotePathID: id,
				Type:       "account",
				Error:      sendErr,
			})
			return posts, fmt.Errorf("failed to EnqueueSendTelegramAccountPost for note_path_id %d: %w", id, sendErr)
		}

		posts = append(posts, ResultPost{
			NotePathID: id,
			Type:       "account",
		})

		noteView := nvs.GetByPathID(id)
		if noteView != nil {
			for inLink := range noteView.InLinks {
				inNote, ok := nvs.Map[inLink]
				if ok && inNote.IsTelegramPublishPost() {
					updateIDs[inNote.PathID] = struct{}{}
				}
			}
		}
	}

	for updateID := range updateIDs {
		err = env.EnqueueUpdateTelegramAccountPost(ctx, updateID)
		if err != nil {
			return posts, fmt.Errorf("failed to EnqueueUpdateTelegramAccountPost for note_path_id %d: %w", updateID, err)
		}
	}

	return posts, nil
}
