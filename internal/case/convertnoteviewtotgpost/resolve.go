package convertnoteviewtotgpost

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/image"
	"trip2g/internal/logger"
	"trip2g/internal/markdownv2"
	"trip2g/internal/model"
)

type SentMessage = db.ListTelegramPublishSentMessagesByChatIDRow

type Env interface {
	LatestNoteViews() *model.NoteViews
	Logger() logger.Logger
	ListTelegramPublishSentMessagesByChatID(ctx context.Context, chatID int64) ([]SentMessage, error)

	TimeLocation() *time.Location
	PublicURL() string
}

func Resolve(ctx context.Context, env Env, source model.TelegramPostSource) (*model.TelegramPost, error) {
	logger := logger.WithPrefix(env.Logger(), "convertnoteviewtotgpost")

	sentMap := make(map[int64]*SentMessage)

	if source.ChatID != 0 {
		sentMsgs, err := env.ListTelegramPublishSentMessagesByChatID(ctx, source.ChatID)
		if err != nil {
			return nil, fmt.Errorf("failed to list all telegram publish sent messages: %w", err)
		}

		for _, msg := range sentMsgs {
			sentMap[msg.NotePathID] = &msg
		}
	}

	nvs := env.LatestNoteViews()

	// allowExternalLinks := getAllowExternalLinks(source.NoteView) || source.Instant
	// always allow external links
	allowExternalLinks := true

	publicURL := env.PublicURL()
	post := model.TelegramPost{}

	tr := markdownv2.HTMLConverter{}
	tr.SetLinkResolver(func(target string) (*markdownv2.LinkResolverResult, error) {
		post.LinkCount++

		linkedNV, ok := nvs.Map[target]
		if !ok {
			post.UnresolvedLinkCount++
			return &markdownv2.LinkResolverResult{URL: publicURL}, nil
			// return "", fmt.Errorf("note not found for target: %s", target)
		}

		msg, ok := sentMap[linkedNV.PathID]
		if ok {
			// Already published - return telegram link
			chatID := strings.TrimPrefix(strconv.FormatInt(msg.TelegramChatID, 10), "-100")
			url := fmt.Sprintf("https://t.me/c/%s/%d", chatID, msg.MessageID)
			return &markdownv2.LinkResolverResult{URL: url}, nil
		}

		// Not published yet
		if linkedNV.IsTelegramPublishPost() {
			publishAt, hasPublishAt := linkedNV.ExtractTelegramPublishAt(env.TimeLocation())
			if hasPublishAt {
				// Return unpublished link with publish date
				return &markdownv2.LinkResolverResult{
					Label:     linkedNV.Title,
					PublishAt: &publishAt,
				}, nil
			}
			return &markdownv2.LinkResolverResult{}, nil
		}

		if allowExternalLinks {
			if publicURL == "" {
				logger.Warn("public URL is not set, cannot generate external link")
				return &markdownv2.LinkResolverResult{}, nil
			}

			post.ExternalLinkCount++

			externalURL := publicURL + linkedNV.Permalink
			return &markdownv2.LinkResolverResult{URL: externalURL}, nil
		}

		return nil, fmt.Errorf("note not published: %s", target)
	})

	res := tr.Process(source.NoteView)

	post.Content = res.Content
	post.Warnings = res.Warnings

	firstImageURL := getFirstImageURL(source.NoteView)
	if firstImageURL != nil {
		post.Images = append(post.Images, *firstImageURL)
	}

	// Validate content length limits
	// Telegram limits: 4096 chars for text-only messages, 1024 chars for photo captions
	maxLength := 4096
	if len(post.Images) > 0 {
		maxLength = 1024
	}

	contentLength := len(post.Content)
	if contentLength > maxLength {
		msgType := "text message"
		if len(post.Images) > 0 {
			msgType = "photo caption"
		}
		post.Warnings = append(post.Warnings, fmt.Sprintf("telegram %s content exceeds limit: %d characters (max %d)", msgType, contentLength, maxLength))
	}

	return &post, nil
}

func getFirstImageURL(noteView *model.NoteView) *string {
	for path := range noteView.Assets {
		if !image.IsRightExtension(path) {
			continue
		}

		_, ok := noteView.AssetReplaces[path]
		if !ok {
			continue
		}

		return &noteView.AssetReplaces[path].URL
	}

	return nil
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
