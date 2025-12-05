package importtelegramchannel

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gotd/td/tg"

	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/tgtd"
)

const fetchBatchSize = 100

type Env interface {
	Logger() logger.Logger
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
	TelegramClientForAccount(account db.TelegramAccount) *tgtd.Client
	// PushNotes saves notes to the database.
	PushNotes(ctx context.Context, input graphmodel.PushNotesInput) (graphmodel.PushNotesOrErrorPayload, error)
	// UploadNoteAsset uploads an asset for a note.
	UploadNoteAsset(ctx context.Context, input graphmodel.UploadNoteAssetInput) (graphmodel.UploadNoteAssetOrErrorPayload, error)
}

type Result struct {
	ImportedCount int
	AssetsCount   int
	Errors        []string
}

// messageGroup represents a single post (may contain multiple messages if it's a media group)
type messageGroup struct {
	primaryMsg *tg.Message   // First message in group (has text/caption)
	allMsgs    []*tg.Message // All messages in the group
}

// messageInfo stores pre-computed info for two-pass processing
type messageInfo struct {
	group    *messageGroup
	title    string
	filename string
	skip     bool
}

// groupMessagesByMediaGroup groups messages by GroupedID into logical posts
func groupMessagesByMediaGroup(messages []*tg.Message) []*messageGroup {
	// Map groupedID -> messages
	groupMap := make(map[int64][]*tg.Message)
	var ungrouped []*tg.Message

	for _, msg := range messages {
		groupedID, ok := msg.GetGroupedID()
		if ok && groupedID != 0 {
			groupMap[groupedID] = append(groupMap[groupedID], msg)
		} else {
			ungrouped = append(ungrouped, msg)
		}
	}

	var result []*messageGroup

	// Add grouped messages
	for _, msgs := range groupMap {
		// Find primary message (one with text, or first one)
		var primary *tg.Message
		for _, m := range msgs {
			if m.Message != "" {
				primary = m
				break
			}
		}
		if primary == nil {
			primary = msgs[0]
		}

		result = append(result, &messageGroup{
			primaryMsg: primary,
			allMsgs:    msgs,
		})
	}

	// Add ungrouped messages
	for _, msg := range ungrouped {
		result = append(result, &messageGroup{
			primaryMsg: msg,
			allMsgs:    []*tg.Message{msg},
		})
	}

	return result
}

// sortGroupsByID sorts groups by primary message ID (descending - newest first)
func sortGroupsByID(groups []*messageGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].primaryMsg.ID > groups[j].primaryMsg.ID
	})
}

func Resolve(ctx context.Context, env Env, params model.ImportTelegramChannelParams) error {
	log := logger.WithPrefix(env.Logger(), "importtelegramchannel:")

	result := &Result{
		Errors: []string{},
	}

	// Get telegram account
	account, err := env.GetTelegramAccountByID(ctx, params.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get telegram account: %w", err)
	}

	tgClient := env.TelegramClientForAccount(account)

	// PHASE 1: Fetch ALL messages (metadata only, no media download)
	var allMessages []*tg.Message

	err = tgClient.RunWithAPI(ctx, account.SessionData, func(ctx context.Context, api *tg.Client) error {
		offsetID := 0

		for {
			msgResult, fetchErr := tgClient.GetChannelMessagesWithAPI(ctx, api, tgtd.GetChannelMessagesParams{
				ChannelID: params.ChannelID,
				Limit:     fetchBatchSize,
				OffsetID:  offsetID,
			})
			if fetchErr != nil {
				return fmt.Errorf("failed to fetch messages: %w", fetchErr)
			}

			if len(msgResult.Messages) == 0 {
				break
			}

			allMessages = append(allMessages, msgResult.Messages...)
			log.Info("fetched messages batch", "count", len(msgResult.Messages), "total", len(allMessages))

			offsetID = msgResult.Messages[len(msgResult.Messages)-1].ID

			if !msgResult.HasMore {
				break
			}
		}

		log.Info("total messages fetched", "count", len(allMessages))
		return nil
	})

	if err != nil {
		return fmt.Errorf("telegram API error: %w", err)
	}

	// Group messages by media group
	groups := groupMessagesByMediaGroup(allMessages)
	log.Info("grouped messages", "totalMessages", len(allMessages), "groups", len(groups))

	// Sort groups by primary message ID (descending, newest first - will reverse later)
	sortGroupsByID(groups)

	// PHASE 2: Build complete postMap (process oldest first for correct order)
	usedFilenames := make(map[string]bool)
	postMap := make(map[string]string) // messageID -> title
	messageInfos := make([]messageInfo, len(groups))

	// Process in reverse order (oldest first)
	for i := len(groups) - 1; i >= 0; i-- {
		group := groups[i]
		idx := len(groups) - 1 - i
		msg := group.primaryMsg

		// Convert and extract title from primary message
		markdown := tgtd.Convert(msg)
		title := extractTitle(markdown)
		if title == "" {
			title = fmt.Sprintf("message-%d", msg.ID)
		}

		// Generate unique filename
		filename := generateFilename(title, msg.ID, usedFilenames)
		usedFilenames[filename] = true

		// Store in postMap for wikilink resolution (all message IDs in group point to same note)
		titleWithoutExt := strings.TrimSuffix(filename, ".md")
		for _, m := range group.allMsgs {
			postMap[strconv.Itoa(m.ID)] = titleWithoutExt
		}

		messageInfos[idx] = messageInfo{
			group:    group,
			title:    titleWithoutExt,
			filename: filename,
			skip:     false,
		}
	}

	log.Info("pass 1 complete", "postMapSize", len(postMap), "toImport", len(groups))

	// PHASE 3: Create notes with full postMap (wikilinks resolved) and upload assets
	assetsDir := fmt.Sprintf("%s/assets", params.BasePath)

	for _, info := range messageInfos {
		if info.skip {
			continue
		}

		msg := info.group.primaryMsg

		// Download media for this group only
		var groupMedia []tgtd.DownloadedMedia
		hasMedia := false
		for _, m := range info.group.allMsgs {
			if m.Media != nil {
				hasMedia = true
				break
			}
		}

		if hasMedia {
			downloadErr := tgClient.RunWithAPI(ctx, account.SessionData, func(ctx context.Context, api *tg.Client) error {
				for _, m := range info.group.allMsgs {
					if m.Media == nil {
						continue
					}
					media, err := tgtd.DownloadMessageMedia(ctx, api, m)
					if err != nil {
						log.Warn("failed to download media", "msgID", m.ID, "error", err)
						continue
					}
					groupMedia = append(groupMedia, media...)
				}
				return nil
			})
			if downloadErr != nil {
				log.Warn("failed to download media for group", "msgID", msg.ID, "error", downloadErr)
			}
		}

		// Convert message to markdown
		markdown := tgtd.Convert(msg)

		// Replace telegram links with wikilinks (using COMPLETE postMap)
		markdown = replaceTelegramLinks(markdown, postMap)

		// Build asset links and prepare filenames
		var assetLinks []string
		var assetInfos []assetInfo

		for idx, media := range groupMedia {
			ext := filepath.Ext(media.Filename)
			assetFilename := fmt.Sprintf("%d_%d%s", msg.ID, idx, ext)
			// Relative path with ./ prefix (used for both markdown and upload)
			relativePath := fmt.Sprintf("./assets/%s", assetFilename)
			absolutePath := fmt.Sprintf("/%s/%s", assetsDir, assetFilename)

			assetInfos = append(assetInfos, assetInfo{
				media:        &groupMedia[idx],
				relativePath: relativePath,
				absolutePath: absolutePath,
				filename:     assetFilename,
			})

			// Add markdown image link (parsed as asset by the system)
			assetLinks = append(assetLinks, fmt.Sprintf("![%s](%s)", assetFilename, relativePath))
		}

		// Prepend asset links to markdown
		if len(assetLinks) > 0 {
			assetSection := strings.Join(assetLinks, "\n") + "\n\n"
			markdown = assetSection + markdown
		}

		// Generate frontmatter
		frontmatter := generateFrontmatter(params.ChannelID, msg)

		// Full note content
		content := frontmatter + markdown

		// Full path
		notePath := fmt.Sprintf("%s/%s", params.BasePath, info.filename)

		// Push single note
		pushInput := graphmodel.PushNotesInput{
			Updates: []graphmodel.PushNoteInput{
				{
					Path:    notePath,
					Content: content,
				},
			},
		}

		payload, pushErr := env.PushNotes(ctx, pushInput)
		if pushErr != nil {
			errMsg := fmt.Sprintf("failed to push note %s: %v", notePath, pushErr)
			result.Errors = append(result.Errors, errMsg)
			log.Warn(errMsg)
			// Cleanup media for this message
			for i := range groupMedia {
				groupMedia[i].Cleanup()
			}
			continue
		}

		switch p := payload.(type) {
		case *graphmodel.ErrorPayload:
			errMsg := fmt.Sprintf("push note error %s: %s", notePath, p.Message)
			result.Errors = append(result.Errors, errMsg)
			log.Warn(errMsg)
			// Cleanup media for this message
			for i := range groupMedia {
				groupMedia[i].Cleanup()
			}
			continue
		case *graphmodel.PushNotesPayload:
			// Find our note by path
			var note *graphmodel.PushedNote
			for i := range p.Notes {
				if p.Notes[i].Path == notePath {
					note = &p.Notes[i]
					break
				}
			}
			if note == nil {
				log.Warn("note not found in response", "path", notePath)
				continue
			}

			log.Info("pushed note",
				"noteID", note.ID,
				"path", note.Path,
				"assetsCount", len(note.Assets))
			for _, asset := range note.Assets {
				log.Info("note asset", "path", asset.Path)
			}

			noteID := note.ID

			// Upload assets for this note
			for _, asset := range assetInfos {
				log.Info("uploading asset", "relativePath", asset.relativePath, "absolutePath", asset.absolutePath)
				uploadErr := uploadAsset(ctx, env, log, noteID, asset)
				if uploadErr != nil {
					errMsg := fmt.Sprintf("failed to upload asset %s: %v", asset.filename, uploadErr)
					result.Errors = append(result.Errors, errMsg)
					log.Warn(errMsg)
				} else {
					result.AssetsCount++
				}
				// Cleanup temp file after upload attempt
				asset.media.Cleanup()
			}

			result.ImportedCount++
			log.Debug("imported message", "id", msg.ID, "path", notePath, "assets", len(assetInfos), "groupSize", len(info.group.allMsgs))
		}
	}

	log.Info("import completed",
		"imported", result.ImportedCount,
		"assets", result.AssetsCount,
		"errors", len(result.Errors))

	return nil
}

type assetInfo struct {
	media        *tgtd.DownloadedMedia
	relativePath string // "./assets/filename"
	absolutePath string
	filename     string
}

func uploadAsset(ctx context.Context, env Env, log logger.Logger, noteID int64, asset assetInfo) error {
	// Open temp file for reading
	file, err := asset.media.Open()
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer file.Close()

	input := graphmodel.UploadNoteAssetInput{
		NoteID:       noteID,
		Path:         asset.relativePath,
		Sha256Hash:   asset.media.Sha256Hash,
		AbsolutePath: asset.absolutePath,
		File: graphql.Upload{
			File:        file,
			Filename:    asset.filename,
			Size:        asset.media.Size,
			ContentType: asset.media.MimeType,
		},
	}

	payload, err := env.UploadNoteAsset(ctx, input)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	switch p := payload.(type) {
	case *graphmodel.ErrorPayload:
		return fmt.Errorf("upload error: %s", p.Message)
	case *graphmodel.UploadNoteAssetPayload:
		if p.UploadSkipped {
			log.Debug("asset upload skipped (already exists)", "path", asset.relativePath)
		}
	}

	return nil
}

// buildPostMapFromNotes builds a message_id -> title map from existing notes.
func buildPostMapFromNotes(nvs *model.NoteViews, channelID int64) map[string]string {
	postMap := make(map[string]string)
	for _, note := range nvs.List {
		noteChannelID, hasChannel := note.ExtractTelegramPublishChannelID()
		messageID, hasMessage := note.ExtractTelegramPublishMessageID()
		if hasChannel && hasMessage && noteChannelID == channelID {
			// Extract title from path (filename without .md)
			parts := strings.Split(note.Path, "/")
			filename := parts[len(parts)-1]
			title := strings.TrimSuffix(filename, ".md")
			postMap[strconv.Itoa(messageID)] = title
		}
	}
	return postMap
}

// buildUsedFilenamesFromNotes builds a set of used filenames in the target directory.
func buildUsedFilenamesFromNotes(nvs *model.NoteViews, basePath string) map[string]bool {
	usedFilenames := make(map[string]bool)
	prefix := basePath + "/"
	for _, note := range nvs.List {
		if strings.HasPrefix(note.Path, prefix) {
			// Extract filename
			filename := strings.TrimPrefix(note.Path, prefix)
			if !strings.Contains(filename, "/") {
				usedFilenames[filename] = true
			}
		}
	}
	return usedFilenames
}

func generateFrontmatter(channelID int64, msg *tg.Message) string {
	publishAt := time.Unix(int64(msg.Date), 0).Format(time.RFC3339)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("telegram_publish_channel_id: \"%d\"\n", channelID))
	sb.WriteString(fmt.Sprintf("telegram_publish_message_id: %d\n", msg.ID))
	sb.WriteString(fmt.Sprintf("telegram_publish_at: %s\n", publishAt))
	sb.WriteString("---\n\n")
	return sb.String()
}
