package convertnoteviewtotgpost

import (
	"context"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/markdownv2"
	"trip2g/internal/model"
)

type Env interface {
	LatestNoteViews() *model.NoteViews
	Logger() logger.Logger
	ListAllTelegramPublishSentMessages(ctx context.Context) ([]db.ListAllTelegramPublishSentMessagesRow, error)
}

func Resolve(ctx context.Context, env Env, nv *model.NoteView) (*model.TelegramPost, error) {
	// logger := logger.WithPrefix(env.Logger(), "convertnoteviewtotgpost")

	sentMsgs, err := env.ListAllTelegramPublishSentMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all telegram publish sent messages: %w", err)
	}

	sentMap := make(map[int64]*db.ListAllTelegramPublishSentMessagesRow)

	for _, msg := range sentMsgs {
		sentMap[msg.NotePathID] = &msg
	}

	nvs := env.LatestNoteViews()

	tr := markdownv2.HTMLConverter{}
	tr.SetLinkResolver(func(target string) (string, error) {
		linkedNV, ok := nvs.Map[target]
		if !ok {
			return "", fmt.Errorf("note not found for target: %s", target)
		}

		msg, ok := sentMap[linkedNV.PathID]
		if !ok {
			return "", fmt.Errorf("note not published: %s", target)
		}

		// remove -100 prefix
		chatID := strings.TrimPrefix(fmt.Sprintf("%d", msg.TelegramChatID), "-100")
		url := fmt.Sprintf("https://t.me/c/%s/%d", chatID, msg.MessageID)

		return url, nil
	})

	res := tr.Process(nv)

	return &model.TelegramPost{
		Content:  res.Content,
		Warnings: res.Warnings,
	}, nil
}
