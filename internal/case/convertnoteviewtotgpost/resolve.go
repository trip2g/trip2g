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
	"trip2g/internal/telegram"
)

// ErrAssetsNotReadyError indicates that media assets are not yet uploaded.
type ErrAssetsNotReadyError struct {
	MissingAssets []string
}

func (e *ErrAssetsNotReadyError) Error() string {
	return fmt.Sprintf("media assets not yet uploaded: %v", e.MissingAssets)
}

type SentMessage = db.ListTelegramPublishSentMessagesByChatIDRow

type Env interface {
	LatestNoteViews() *model.NoteViews
	Logger() logger.Logger
	ListTelegramPublishSentMessagesByChatID(ctx context.Context, chatID int64) ([]SentMessage, error)

	TimeLocation() *time.Location
	PublicURL() string
	Now() time.Time
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
				now := env.Now()
				threshold := now.Add(30 * time.Minute)

				// If publish time is in the past or within 30 minutes, render as underlined text without footer
				if publishAt.Before(threshold) || publishAt.Equal(threshold) {
					return &markdownv2.LinkResolverResult{
						Label: linkedNV.Title,
					}, nil
				}

				// Publish time is more than 30 minutes away - add to footer
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

	mediaURLs, err := getAllMediaURLs(source.NoteView)
	if err != nil {
		return nil, err
	}
	post.Media = mediaURLs

	// Validate content length limits
	// Telegram limits: 4096 chars for text-only messages, 1024 chars for photo captions
	// Telegram counts visible length (without HTML tags)
	maxLength := 4096
	if len(post.Media) > 0 {
		maxLength = 1024
	}

	contentLength := telegram.GetVisibleTelegramLength(post.Content)
	if contentLength > maxLength {
		msgType := "text message"
		if len(post.Media) > 0 {
			msgType = "photo caption"
		}
		post.Warnings = append(post.Warnings, fmt.Sprintf("telegram %s content exceeds limit: %d characters (max %d)", msgType, contentLength, maxLength))
	}

	// Telegram media group limit is 10
	if len(post.Media) > 10 {
		warning := fmt.Sprintf(
			"telegram media group limit exceeded: %d files (max 10, only first 10 will be used)",
			len(post.Media),
		)
		post.Warnings = append(post.Warnings, warning)
		post.Media = post.Media[:10]
	}

	return &post, nil
}

func getAllMediaURLs(noteView *model.NoteView) ([]string, error) {
	var mediaURLs []string
	var missingAssets []string

	for path := range noteView.Assets {
		if !image.IsMediaExtension(path) {
			continue
		}

		assetReplace, ok := noteView.AssetReplaces[path]
		if !ok {
			missingAssets = append(missingAssets, path)
			continue
		}

		mediaURLs = append(mediaURLs, assetReplace.URL)

		// Telegram allows max 10 media files in a group
		if len(mediaURLs) >= 10 {
			break
		}
	}

	if len(missingAssets) > 0 {
		return nil, &ErrAssetsNotReadyError{MissingAssets: missingAssets}
	}

	return mediaURLs, nil
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
