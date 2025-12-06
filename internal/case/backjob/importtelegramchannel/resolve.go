package importtelegramchannel

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gotd/td/tg"
	"golang.org/x/sync/errgroup"

	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/tgtd"
)

const (
	fetchBatchSize = 100
	pushBatchSize  = 10
	assetWorkers   = 5
)

type Env interface {
	Logger() logger.Logger
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
	TelegramClientForAccount(account db.TelegramAccount) *tgtd.Client
	LatestNoteViews() *model.NoteViews
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

	// Build map of existing imported notes
	nvs := env.LatestNoteViews()
	importedNotes := model.BuildImportedNotesMap(nvs)
	log.Info("loaded existing notes", "count", len(importedNotes))

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

	// Sort groups by primary message ID (descending, newest first)
	sortGroupsByID(groups)

	// Partition: not-imported first, then already-imported (both newest to oldest)
	var notImported, alreadyImported []*messageGroup
	for _, g := range groups {
		key := model.FormatImportKey(params.ChannelID, g.primaryMsg.ID)
		if _, exists := importedNotes[key]; exists {
			alreadyImported = append(alreadyImported, g)
		} else {
			notImported = append(notImported, g)
		}
	}
	groups = append(notImported, alreadyImported...)
	log.Info("partitioned groups", "notImported", len(notImported), "alreadyImported", len(alreadyImported))

	// PHASE 2: Build complete postMap and messageInfos (newest first, not-imported before already-imported)
	usedFilenames := make(map[string]bool)
	postMap := make(map[string]string) // messageID -> title
	messageInfos := make([]messageInfo, len(groups))

	for i, group := range groups {
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

		messageInfos[i] = messageInfo{
			group:    group,
			title:    titleWithoutExt,
			filename: filename,
			skip:     false,
		}
	}

	log.Info("phase 2 complete", "postMapSize", len(postMap), "toProcess", len(groups))

	// PHASE 3: Create notes with full postMap (wikilinks resolved) and upload assets
	assetsDir := fmt.Sprintf("%s/assets", params.BasePath)

	// Count items to process (for determining last batch)
	toProcessCount := 0
	for _, info := range messageInfos {
		if !info.skip {
			toProcessCount++
		}
	}

	// Process in batches
	var batch []preparedNote
	processedCount := 0

	flushBatch := func(isLastBatch bool) error {
		if len(batch) == 0 {
			return nil
		}

		// Determine if we should rebuild index
		shouldRebuildIndex := (result.ImportedCount+len(batch))%50 < len(batch) || isLastBatch

		// Build push input with all notes in batch
		updates := make([]graphmodel.PushNoteInput, len(batch))
		for i, note := range batch {
			updates[i] = graphmodel.PushNoteInput{
				Path:    note.path,
				Content: note.content,
			}
		}

		pushInput := graphmodel.PushNotesInput{
			Partial: !shouldRebuildIndex,
			Updates: updates,
		}

		if shouldRebuildIndex {
			log.Info("will rebuild search index", "batchSize", len(batch), "totalImported", result.ImportedCount+len(batch))
		}

		log.Info("pushing batch", "size", len(batch))
		payload, pushErr := env.PushNotes(ctx, pushInput)
		if pushErr != nil {
			errMsg := fmt.Sprintf("failed to push batch: %v", pushErr)
			result.Errors = append(result.Errors, errMsg)
			log.Warn(errMsg)
			// Cleanup all media in batch
			for _, note := range batch {
				for _, asset := range note.assets {
					asset.media.Cleanup()
				}
			}
			batch = nil
			return nil
		}

		switch p := payload.(type) {
		case *graphmodel.ErrorPayload:
			errMsg := fmt.Sprintf("push batch error: %s", p.Message)
			result.Errors = append(result.Errors, errMsg)
			log.Warn(errMsg)
			// Cleanup all media in batch
			for _, note := range batch {
				for _, asset := range note.assets {
					asset.media.Cleanup()
				}
			}
		case *graphmodel.PushNotesPayload:
			// Build path -> noteID map
			noteIDByPath := make(map[string]int64)
			for _, n := range p.Notes {
				noteIDByPath[n.Path] = n.ID
			}

			// Collect all assets to upload with their note IDs
			type assetUploadJob struct {
				noteID int64
				asset  assetInfo
			}
			var jobs []assetUploadJob

			for _, note := range batch {
				noteID, ok := noteIDByPath[note.path]
				if !ok {
					log.Warn("note not found in response", "path", note.path)
					for _, asset := range note.assets {
						asset.media.Cleanup()
					}
					continue
				}

				for _, asset := range note.assets {
					jobs = append(jobs, assetUploadJob{noteID: noteID, asset: asset})
				}
				result.ImportedCount++
			}

			// Upload assets in parallel
			if len(jobs) > 0 {
				log.Info("uploading assets", "count", len(jobs), "workers", assetWorkers)
				uploadStart := time.Now()

				var mu sync.Mutex
				g, gctx := errgroup.WithContext(ctx)
				g.SetLimit(assetWorkers)

				for _, job := range jobs {
					job := job // capture
					g.Go(func() error {
						uploadErr := uploadAsset(gctx, env, log, job.noteID, job.asset)
						job.asset.media.Cleanup()

						mu.Lock()
						defer mu.Unlock()

						if uploadErr != nil {
							errMsg := fmt.Sprintf("failed to upload asset %s: %v", job.asset.filename, uploadErr)
							result.Errors = append(result.Errors, errMsg)
							log.Warn(errMsg)
						} else {
							result.AssetsCount++
						}
						return nil // don't fail the group on asset errors
					})
				}
				g.Wait()

				log.Info("assets uploaded", "count", result.AssetsCount, "duration", time.Since(uploadStart).Round(time.Millisecond))
			}
		}

		batch = nil
		return nil
	}

	for _, info := range messageInfos {
		if info.skip {
			continue
		}
		processedCount++
		isLast := processedCount == toProcessCount

		msg := info.group.primaryMsg

		// Download media for this group
		var groupMedia []tgtd.DownloadedMedia
		hasMedia := false
		for _, m := range info.group.allMsgs {
			if m.Media != nil {
				hasMedia = true
				break
			}
		}

		if hasMedia {
			downloadStart := time.Now()
			var totalSize int64
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
					for _, med := range media {
						totalSize += med.Size
					}
					groupMedia = append(groupMedia, media...)
				}
				return nil
			})
			if downloadErr != nil {
				log.Warn("failed to download media for group", "msgID", msg.ID, "error", downloadErr)
			} else if len(groupMedia) > 0 {
				log.Info("downloaded media",
					"msgID", msg.ID,
					"count", len(groupMedia),
					"totalSize", formatBytes(totalSize),
					"duration", time.Since(downloadStart).Round(time.Millisecond))
			}
		}

		// Convert message to markdown
		markdown := tgtd.Convert(msg)

		// Replace telegram links with wikilinks
		markdown = replaceTelegramLinks(markdown, postMap)

		// Build asset links and prepare filenames
		var assetLinks []string
		var assets []assetInfo

		for idx, media := range groupMedia {
			ext := filepath.Ext(media.Filename)
			assetFilename := fmt.Sprintf("%d_%d%s", msg.ID, idx, ext)
			relativePath := fmt.Sprintf("./assets/%s", assetFilename)
			absolutePath := fmt.Sprintf("/%s/%s", assetsDir, assetFilename)

			assets = append(assets, assetInfo{
				media:        &groupMedia[idx],
				relativePath: relativePath,
				absolutePath: absolutePath,
				filename:     assetFilename,
			})

			assetLinks = append(assetLinks, fmt.Sprintf("![%s](%s)", assetFilename, relativePath))
		}

		// Prepend asset links to markdown
		if len(assetLinks) > 0 {
			assetSection := strings.Join(assetLinks, "\n") + "\n\n"
			markdown = assetSection + markdown
		}

		// Generate frontmatter and full content
		frontmatter := generateFrontmatter(params.ChannelID, msg)
		content := frontmatter + markdown
		notePath := fmt.Sprintf("%s/%s", params.BasePath, info.filename)

		// Add to batch
		batch = append(batch, preparedNote{
			path:    notePath,
			content: content,
			assets:  assets,
			msgID:   msg.ID,
		})

		// Flush batch if full or last
		if len(batch) >= pushBatchSize || isLast {
			flushBatch(isLast)
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

// preparedNote holds all data needed to push a note and upload its assets
type preparedNote struct {
	path    string
	content string
	assets  []assetInfo
	msgID   int
}

func uploadAsset(ctx context.Context, env Env, log logger.Logger, noteID int64, asset assetInfo) error {
	// Open temp file for reading
	file, err := asset.media.Open()
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer file.Close()

	input := graphmodel.UploadNoteAssetInput{
		Partial:      true,
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
	messageLink := fmt.Sprintf("https://t.me/c/%d/%d", channelID, msg.ID)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("telegram_publish_channel_id: \"%d\"\n", channelID))
	sb.WriteString(fmt.Sprintf("telegram_publish_message_id: %d\n", msg.ID))
	sb.WriteString(fmt.Sprintf("telegram_publish_message_link: %s\n", messageLink))
	sb.WriteString(fmt.Sprintf("telegram_publish_at: %s\n", publishAt))
	sb.WriteString("---\n\n")
	return sb.String()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
