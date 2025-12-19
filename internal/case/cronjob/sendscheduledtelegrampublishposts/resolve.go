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

type jobConfig struct {
	logPrefix     string
	postType      string
	listIDs       func(ctx context.Context) ([]int64, error)
	enqueueSend   func(ctx context.Context, params model.SendTelegramPublishPostParams) error
	enqueueUpdate func(ctx context.Context, notePathID int64) error
}

func enqueueBotJobs(ctx context.Context, env Env) ([]ResultPost, error) {
	return enqueueJobs(ctx, env, jobConfig{
		logPrefix:     "sendscheduledtelegrampublishposts:bot:",
		postType:      "bot",
		listIDs:       env.ListSheduledTelegarmPublishNoteIDs,
		enqueueSend:   env.EnqueueSendTelegramPost,
		enqueueUpdate: env.EnqueueUpdateTelegramPost,
	})
}

func enqueueAccountJobs(ctx context.Context, env Env) ([]ResultPost, error) {
	return enqueueJobs(ctx, env, jobConfig{
		logPrefix:     "sendscheduledtelegrampublishposts:account:",
		postType:      "account",
		listIDs:       env.ListSheduledTelegarmAccountPublishNoteIDs,
		enqueueSend:   env.EnqueueSendTelegramAccountPost,
		enqueueUpdate: env.EnqueueUpdateTelegramAccountPost,
	})
}

func enqueueJobs(ctx context.Context, env Env, cfg jobConfig) ([]ResultPost, error) {
	log := logger.WithPrefix(env.Logger(), cfg.logPrefix)

	ids, err := cfg.listIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled %s publish note IDs: %w", cfg.postType, err)
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

		sendErr := cfg.enqueueSend(ctx, params)
		if sendErr != nil {
			posts = append(posts, ResultPost{
				NotePathID: id,
				Type:       cfg.postType,
				Error:      sendErr,
			})
			return posts, fmt.Errorf("failed to enqueue send %s post for note_path_id %d: %w", cfg.postType, id, sendErr)
		}

		posts = append(posts, ResultPost{
			NotePathID: id,
			Type:       cfg.postType,
		})

		noteView := nvs.GetByPathID(id)
		if noteView != nil {
			log.Info("checking inLinks",
				"note_path_id", id,
				"permalink", noteView.Permalink,
				"inLinks_count", len(noteView.InLinks),
			)
			for inLink := range noteView.InLinks {
				inNote, ok := nvs.Map[inLink]
				if !ok {
					log.Info("inLink not found in nvs.Map", "inLink", inLink)
					continue
				}
				if !inNote.IsTelegramPublishPost() {
					log.Info("inLink is not a telegram publish post",
						"inLink", inLink,
						"inNote_path_id", inNote.PathID,
					)
					continue
				}
				log.Info("adding to updateIDs",
					"inLink", inLink,
					"inNote_path_id", inNote.PathID,
				)
				updateIDs[inNote.PathID] = struct{}{}
			}
		} else {
			log.Info("noteView not found", "note_path_id", id)
		}
	}

	log.Info("enqueuing updates", "updateIDs_count", len(updateIDs))
	for updateID := range updateIDs {
		log.Info("enqueuing update", "updateID", updateID)
		err = cfg.enqueueUpdate(ctx, updateID)
		if err != nil {
			return posts, fmt.Errorf("failed to enqueue update %s post for note_path_id %d: %w", cfg.postType, updateID, err)
		}
	}

	return posts, nil
}
