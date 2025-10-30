package convertnoteviewtotgpost

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/markdownv2"
	"trip2g/internal/model"
)

type SentMessage = db.ListTelegramPublishSentMessagesByChatIDRow

type Env interface {
	LatestNoteViews() *model.NoteViews
	Logger() logger.Logger
	ListTelegramPublishSentMessagesByChatID(ctx context.Context, chatID int64) ([]SentMessage, error)
	PublicURL() string
}

func Resolve(ctx context.Context, env Env, source model.TelegramPostSource) (*model.TelegramPost, error) {
	logger := logger.WithPrefix(env.Logger(), "convertnoteviewtotgpost")

	sentMsgs, err := env.ListTelegramPublishSentMessagesByChatID(ctx, source.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to list all telegram publish sent messages: %w", err)
	}

	sentMap := make(map[int64]*SentMessage)

	for _, msg := range sentMsgs {
		sentMap[msg.NotePathID] = &msg
	}

	nvs := env.LatestNoteViews()

	// allowExternalLinks := getAllowExternalLinks(source.NoteView) || source.Instant
	// always allow external links
	allowExternalLinks := true

	publicURL := env.PublicURL()

	tr := markdownv2.HTMLConverter{}
	tr.SetLinkResolver(func(target string) (string, error) {
		linkedNV, ok := nvs.Map[target]
		if !ok {
			return publicURL, nil
			// return "", fmt.Errorf("note not found for target: %s", target)
		}

		msg, ok := sentMap[linkedNV.PathID]
		if !ok {
			if allowExternalLinks {
				if publicURL == "" {
					logger.Warn("public URL is not set, cannot generate external link")
					return "", nil
				}

				externalURL := publicURL + linkedNV.Permalink
				return externalURL, nil
			}

			return "", fmt.Errorf("note not published: %s", target)
		}

		// remove -100 prefix
		chatID := strings.TrimPrefix(strconv.FormatInt(msg.TelegramChatID, 10), "-100")
		url := fmt.Sprintf("https://t.me/c/%s/%d", chatID, msg.MessageID)

		return url, nil
	})

	res := tr.Process(source.NoteView)

	return &model.TelegramPost{
		Content:  res.Content,
		Warnings: res.Warnings,
	}, nil
}

/*
func getAllowExternalLinks(nv *model.NoteView) bool {
	val, ok := nv.RawMeta["telegram_publish_allow_external_links"]
	if ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true"
		}
	}

	return false
}
*/
