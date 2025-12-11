# Telegram Channel Import Implementation Plan

## Overview

Import posts from a Telegram channel and create notes for each post. This feature allows importing historical content from Telegram channels into the notes system.

**Architecture**: Uses async background job pattern to avoid long transactions and `SQLITE_BUSY` errors.

## GraphQL Schema

```graphql
input AdminImportTelegramAccountChannelInput {
  accountId: Int64!      # Telegram account ID from database
  channelId: Int64!      # Telegram channel ID (numeric)
  basePath: String!      # Target folder path, e.g. "imported/channel"
}

type AdminImportTelegramAccountChannelPayload {
  success: Boolean!
  jobId: String!         # ID of enqueued background job
}

union AdminImportTelegramAccountChannelOrErrorPayload =
  AdminImportTelegramAccountChannelPayload | ErrorPayload
```

Add to `AdminMutation` (Telegram account mutations section):
```graphql
importTelegramAccountChannel(input: AdminImportTelegramAccountChannelInput!): AdminImportTelegramAccountChannelOrErrorPayload!
```

## Architecture

The import is split into two layers:

### Layer 1: GraphQL Mutation (Synchronous)
- **Location**: `internal/case/admin/importtelegramaccountchannel/`
- **Purpose**: Validate input, sanitize basePath, enqueue background job
- **Duration**: Fast (no I/O operations)

### Layer 2: Background Job (Asynchronous)
- **Location**: `internal/case/backjob/importtelegramchannel/`
- **Purpose**: Fetch messages from Telegram, convert, save notes
- **Queue**: `telegramTaskQueue` (`tg_task_jobs`)

> **IMPORTANT**: Do NOT use `tg_api_jobs` queue! That queue has limit=1 and is used for bot messages/notifications. Import can take 10-30 minutes and would block all notifications. Use `tg_task_jobs` instead - tgtd client has its own rate limiting via gotd, and uses user account (not bot), so no conflicts.

## Implementation Steps

### 1. Add GetChannelMessages to tgtd.Client

**File**: `internal/tgtd/client.go`

Add new method to fetch channel messages with pagination:

```go
type GetChannelMessagesParams struct {
    ChannelID int64
    Limit     int   // Max messages to fetch per batch (max 100)
    OffsetID  int   // Message ID to start from (for pagination, 0 = from latest)
}

type GetChannelMessagesResult struct {
    Messages []*tg.Message
    HasMore  bool
}

func (c *Client) GetChannelMessages(ctx context.Context, sessionData []byte, params GetChannelMessagesParams) (*GetChannelMessagesResult, error) {
    storage := &session.StorageMemory{}
    err := storage.StoreSession(ctx, sessionData)
    if err != nil {
        return nil, fmt.Errorf("failed to load session: %w", err)
    }

    client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
        SessionStorage: storage,
    })

    var result *GetChannelMessagesResult

    err = client.Run(ctx, func(ctx context.Context) error {
        api := client.API()

        // Resolve channel peer
        peer, peerErr := c.resolvePeer(ctx, api, params.ChannelID)
        if peerErr != nil {
            return fmt.Errorf("failed to resolve peer: %w", peerErr)
        }

        channelPeer, ok := peer.(*tg.InputPeerChannel)
        if !ok {
            return fmt.Errorf("expected channel peer, got %T", peer)
        }

        limit := params.Limit
        if limit <= 0 || limit > 100 {
            limit = 100
        }

        messages, msgErr := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
            Peer:     channelPeer,
            Limit:    limit,
            OffsetID: params.OffsetID,
        })
        if msgErr != nil {
            return fmt.Errorf("failed to get history: %w", msgErr)
        }

        var msgList []*tg.Message
        var rawCount int  // Count BEFORE filtering

        switch m := messages.(type) {
        case *tg.MessagesChannelMessages:
            rawCount = len(m.Messages)  // Save raw count
            for _, msg := range m.Messages {
                if message, msgOk := msg.(*tg.Message); msgOk {
                    if message.Message != "" { // Skip empty messages
                        msgList = append(msgList, message)
                    }
                }
            }
        case *tg.MessagesMessages:
            rawCount = len(m.Messages)
            for _, msg := range m.Messages {
                if message, msgOk := msg.(*tg.Message); msgOk {
                    if message.Message != "" {
                        msgList = append(msgList, message)
                    }
                }
            }
        case *tg.MessagesMessagesSlice:
            rawCount = len(m.Messages)
            for _, msg := range m.Messages {
                if message, msgOk := msg.(*tg.Message); msgOk {
                    if message.Message != "" {
                        msgList = append(msgList, message)
                    }
                }
            }
        }

        result = &GetChannelMessagesResult{
            Messages: msgList,
            HasMore:  rawCount >= limit,  // Use rawCount, not filtered len!
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return result, nil
}
```

> **IMPORTANT Pagination Fix**: Use `rawCount >= limit` instead of `len(msgList) == limit`. If Telegram returns 100 messages but 5 are media-only (empty text), `len(msgList)` would be 95, and `95 == 100` would be false, stopping pagination prematurely.

### 2. Add RunWithAPI and Media Download Helpers to tgtd.Client

**File**: `internal/tgtd/client.go`

> **CRITICAL**: Do NOT create a new connection for each media download! This would cause:
> - FloodWait errors from Telegram
> - Potential account ban
> - Very slow performance (handshake for each connection)
>
> Instead, use a single connection for the entire import process.

Add `RunWithAPI` wrapper method and helper types:

```go
import (
    "github.com/gotd/td/telegram/downloader"
)

// DownloadedMedia represents downloaded media metadata (not the data itself!)
// The actual data is streamed directly to storage to avoid OOM
type DownloadedMedia struct {
    Filename string
    MimeType string
    Size     int64
}

// MediaDownloadFunc is called for each media item to stream data directly to storage
// This avoids loading large files (videos can be 1.5GB+) into memory
type MediaDownloadFunc func(filename string, mimeType string, reader io.Reader) error

// APIFunc is a function that receives the Telegram API client
type APIFunc func(ctx context.Context, api *tg.Client) error

// RunWithAPI runs a function with an active Telegram API connection.
// Use this to perform multiple operations within a single session.
// This avoids creating new connections for each operation, preventing FloodWait.
func (c *Client) RunWithAPI(ctx context.Context, sessionData []byte, f APIFunc) error {
    storage := &session.StorageMemory{}
    err := storage.StoreSession(ctx, sessionData)
    if err != nil {
        return fmt.Errorf("failed to load session: %w", err)
    }

    client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
        SessionStorage: storage,
    })

    return client.Run(ctx, func(ctx context.Context) error {
        return f(ctx, client.API())
    })
}

// DownloadMessageMediaStreaming downloads media from a message and streams it directly to storage.
// Uses streaming to avoid loading large files (1.5GB+ videos) into memory.
// IMPORTANT: Use this within RunWithAPI callback, not standalone!
//
// The onMedia callback is called for each media item with an io.Reader that streams
// the data directly from Telegram. The callback should write to storage immediately.
func DownloadMessageMediaStreaming(
    ctx context.Context,
    api *tg.Client,
    msg *tg.Message,
    onMedia MediaDownloadFunc,
) ([]DownloadedMedia, error) {
    if msg.Media == nil {
        return nil, nil
    }

    d := downloader.NewDownloader()
    var result []DownloadedMedia

    switch media := msg.Media.(type) {
    case *tg.MessageMediaPhoto:
        if media.Photo == nil {
            return nil, nil
        }
        photo, ok := media.Photo.(*tg.Photo)
        if !ok {
            return nil, nil
        }

        // Get largest photo size
        var bestSize tg.PhotoSizeClass
        var bestWidth int
        for _, size := range photo.Sizes {
            switch s := size.(type) {
            case *tg.PhotoSize:
                if s.W > bestWidth {
                    bestWidth = s.W
                    bestSize = s
                }
            case *tg.PhotoSizeProgressive:
                if s.W > bestWidth {
                    bestWidth = s.W
                    bestSize = s
                }
            }
        }

        if bestSize == nil {
            return nil, nil
        }

        // Build input location
        var location tg.InputFileLocationClass
        switch s := bestSize.(type) {
        case *tg.PhotoSize:
            location = &tg.InputPhotoFileLocation{
                ID:            photo.ID,
                AccessHash:    photo.AccessHash,
                FileReference: photo.FileReference,
                ThumbSize:     s.Type,
            }
        case *tg.PhotoSizeProgressive:
            location = &tg.InputPhotoFileLocation{
                ID:            photo.ID,
                AccessHash:    photo.AccessHash,
                FileReference: photo.FileReference,
                ThumbSize:     s.Type,
            }
        }

        filename := fmt.Sprintf("%d_%d.jpg", msg.ID, photo.ID)
        mimeType := "image/jpeg"

        // Use pipe for streaming: downloader writes to pipeWriter,
        // onMedia reads from pipeReader
        pipeReader, pipeWriter := io.Pipe()

        // Download in goroutine, writing to pipe
        var downloadErr error
        var downloadedSize int64
        go func() {
            defer pipeWriter.Close()
            downloadedSize, downloadErr = d.Download(api, location).Stream(ctx, pipeWriter)
            if downloadErr != nil {
                pipeWriter.CloseWithError(downloadErr)
            }
        }()

        // Stream to storage via callback
        err := onMedia(filename, mimeType, pipeReader)
        pipeReader.Close()

        if err != nil {
            return nil, fmt.Errorf("failed to save photo %s: %w", filename, err)
        }
        if downloadErr != nil {
            return nil, fmt.Errorf("failed to download photo: %w", downloadErr)
        }

        result = append(result, DownloadedMedia{
            Filename: filename,
            MimeType: mimeType,
            Size:     downloadedSize,
        })

    case *tg.MessageMediaDocument:
        if media.Document == nil {
            return nil, nil
        }
        doc, ok := media.Document.(*tg.Document)
        if !ok {
            return nil, nil
        }

        // Check if it's a photo/video (supported media types)
        isImage := false
        isVideo := false
        var filename string

        for _, attr := range doc.Attributes {
            switch a := attr.(type) {
            case *tg.DocumentAttributeFilename:
                filename = a.FileName
            case *tg.DocumentAttributeVideo:
                isVideo = true
            case *tg.DocumentAttributeImageSize:
                isImage = true
            }
        }

        // Determine type from MIME if not set
        if strings.HasPrefix(doc.MimeType, "image/") {
            isImage = true
        }
        if strings.HasPrefix(doc.MimeType, "video/") {
            isVideo = true
        }

        if !isImage && !isVideo {
            return nil, nil // Skip non-media documents
        }

        if filename == "" {
            ext := ".bin"
            if isImage {
                ext = ".jpg"
            } else if isVideo {
                ext = ".mp4"
            }
            filename = fmt.Sprintf("%d_%d%s", msg.ID, doc.ID, ext)
        }

        // Build input location
        location := &tg.InputDocumentFileLocation{
            ID:            doc.ID,
            AccessHash:    doc.AccessHash,
            FileReference: doc.FileReference,
        }

        mimeType := doc.MimeType

        // Use pipe for streaming
        pipeReader, pipeWriter := io.Pipe()

        var downloadErr error
        var downloadedSize int64
        go func() {
            defer pipeWriter.Close()
            downloadedSize, downloadErr = d.Download(api, location).Stream(ctx, pipeWriter)
            if downloadErr != nil {
                pipeWriter.CloseWithError(downloadErr)
            }
        }()

        // Stream to storage via callback
        err := onMedia(filename, mimeType, pipeReader)
        pipeReader.Close()

        if err != nil {
            return nil, fmt.Errorf("failed to save document %s: %w", filename, err)
        }
        if downloadErr != nil {
            return nil, fmt.Errorf("failed to download document: %w", downloadErr)
        }

        result = append(result, DownloadedMedia{
            Filename: filename,
            MimeType: mimeType,
            Size:     downloadedSize,
        })
    }

    return result, nil
}

// GetChannelMessagesWithAPI fetches messages using an existing API connection.
// IMPORTANT: Use this within RunWithAPI callback, not standalone!
func (c *Client) GetChannelMessagesWithAPI(ctx context.Context, api *tg.Client, params GetChannelMessagesParams) (*GetChannelMessagesResult, error) {
    // Resolve channel peer
    peer, err := c.resolvePeerWithAPI(ctx, api, params.ChannelID)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve peer: %w", err)
    }

    channelPeer, ok := peer.(*tg.InputPeerChannel)
    if !ok {
        return nil, fmt.Errorf("expected channel peer, got %T", peer)
    }

    limit := params.Limit
    if limit <= 0 || limit > 100 {
        limit = 100
    }

    messages, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
        Peer:     channelPeer,
        Limit:    limit,
        OffsetID: params.OffsetID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get history: %w", err)
    }

    var msgList []*tg.Message
    var rawCount int

    switch m := messages.(type) {
    case *tg.MessagesChannelMessages:
        rawCount = len(m.Messages)
        for _, msg := range m.Messages {
            if message, msgOk := msg.(*tg.Message); msgOk {
                // Include messages with text OR media
                if message.Message != "" || message.Media != nil {
                    msgList = append(msgList, message)
                }
            }
        }
    case *tg.MessagesMessages:
        rawCount = len(m.Messages)
        for _, msg := range m.Messages {
            if message, msgOk := msg.(*tg.Message); msgOk {
                if message.Message != "" || message.Media != nil {
                    msgList = append(msgList, message)
                }
            }
        }
    case *tg.MessagesMessagesSlice:
        rawCount = len(m.Messages)
        for _, msg := range m.Messages {
            if message, msgOk := msg.(*tg.Message); msgOk {
                if message.Message != "" || message.Media != nil {
                    msgList = append(msgList, message)
                }
            }
        }
    }

    return &GetChannelMessagesResult{
        Messages: msgList,
        HasMore:  rawCount >= limit,
    }, nil
}
```

> **Note**: `GetChannelMessagesWithAPI` now includes messages with media (even if text is empty), since we want to import photos/videos too.

### 3. Add Note Duplicate Detection Methods

**File**: `internal/model/note_telegram.go`

Add methods to extract telegram publish metadata:

```go
import (
    "fmt"
    "strconv"
)

// ExtractTelegramPublishChannelID returns the channel ID if present in metadata
func (note *NoteView) ExtractTelegramPublishChannelID() (int64, bool) {
    rawChannelID, ok := note.RawMeta["telegram_publish_channel_id"]
    if !ok {
        return 0, false
    }
    switch v := rawChannelID.(type) {
    case string:
        id, err := strconv.ParseInt(v, 10, 64)
        if err != nil {
            return 0, false
        }
        return id, true
    case int64:
        return v, true
    case float64:
        return int64(v), true
    case int:
        return int64(v), true
    }
    return 0, false
}

// ExtractTelegramPublishMessageID returns the message ID if present in metadata
func (note *NoteView) ExtractTelegramPublishMessageID() (int, bool) {
    rawMessageID, ok := note.RawMeta["telegram_publish_message_id"]
    if !ok {
        return 0, false
    }
    switch v := rawMessageID.(type) {
    case int:
        return v, true
    case int64:
        return int(v), true
    case float64:
        return int(v), true
    }
    return 0, false
}

// BuildImportedNotesMap builds a map of imported notes keyed by "channelID:messageID"
func BuildImportedNotesMap(nvs *NoteViews) map[string]*NoteView {
    result := make(map[string]*NoteView)
    for _, note := range nvs.List {
        channelID, hasChannel := note.ExtractTelegramPublishChannelID()
        messageID, hasMessage := note.ExtractTelegramPublishMessageID()
        if hasChannel && hasMessage {
            key := fmt.Sprintf("%d:%d", channelID, messageID)
            result[key] = note
        }
    }
    return result
}
```

### 3. Create Background Job Params Model

**File**: `internal/model/telegram.go`

Add params struct for background job:

```go
// ImportTelegramChannelParams contains parameters for import background job
type ImportTelegramChannelParams struct {
    AccountID int64  `json:"account_id"`
    ChannelID int64  `json:"channel_id"`
    BasePath  string `json:"base_path"`
}
```

### 4. Create GraphQL Case (Layer 1 - Synchronous)

**File**: `internal/case/admin/importtelegramaccountchannel/resolve.go`

This case validates, sanitizes basePath, and enqueues the background job:

```go
package importtelegramaccountchannel

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"

    ozzo "github.com/go-ozzo/ozzo-validation/v4"
    "trip2g/internal/db"
    "trip2g/internal/graph/model"
    appmodel "trip2g/internal/model"
    "trip2g/internal/usertoken"
)

type Env interface {
    CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
    GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
    EnqueueImportTelegramChannel(ctx context.Context, params appmodel.ImportTelegramChannelParams) (string, error)
}

type Input = model.AdminImportTelegramAccountChannelInput
type Payload = model.AdminImportTelegramAccountChannelOrErrorPayload

func validateRequest(input *Input) *model.ErrorPayload {
    return model.NewOzzoError(ozzo.ValidateStruct(input,
        ozzo.Field(&input.AccountID, ozzo.Required),
        ozzo.Field(&input.ChannelID, ozzo.Required),
        ozzo.Field(&input.BasePath, ozzo.Required),
    ))
}

// sanitizeBasePath cleans and validates the base path to prevent path traversal
func sanitizeBasePath(basePath string) (string, error) {
    // Clean the path
    cleaned := filepath.Clean(basePath)

    // Check for path traversal attempts
    if strings.Contains(cleaned, "..") {
        return "", fmt.Errorf("invalid path: contains '..'")
    }

    // Remove leading slash for consistency
    cleaned = strings.TrimPrefix(cleaned, "/")

    // Don't allow empty path after cleaning
    if cleaned == "" || cleaned == "." {
        return "", fmt.Errorf("invalid path: empty after cleaning")
    }

    return cleaned, nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
    // Validate admin access
    _, err := env.CurrentAdminUserToken(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get current user token: %w", err)
    }

    // Validate input
    if errPayload := validateRequest(&input); errPayload != nil {
        return errPayload, nil
    }

    // Sanitize basePath to prevent path traversal
    sanitizedPath, err := sanitizeBasePath(input.BasePath)
    if err != nil {
        return &model.ErrorPayload{Message: err.Error()}, nil
    }

    // Verify account exists
    _, err = env.GetTelegramAccountByID(ctx, input.AccountID)
    if err != nil {
        return &model.ErrorPayload{Message: "Telegram account not found"}, nil
    }

    // Enqueue background job with sanitized path
    params := appmodel.ImportTelegramChannelParams{
        AccountID: input.AccountID,
        ChannelID: input.ChannelID,
        BasePath:  sanitizedPath,
    }

    jobID, err := env.EnqueueImportTelegramChannel(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("failed to enqueue import job: %w", err)
    }

    payload := model.AdminImportTelegramAccountChannelPayload{
        Success: true,
        JobID:   jobID,
    }

    return &payload, nil
}
```

### 5. Create Background Job (Layer 2 - Asynchronous)

**File**: `internal/case/backjob/importtelegramchannel/resolve.go`

Uses **two-pass processing** for bidirectional wikilink resolution and **asset downloading**:

```go
package importtelegramchannel

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/gotd/td/tg"
    "trip2g/internal/db"
    "trip2g/internal/logger"
    appmodel "trip2g/internal/model"
    "trip2g/internal/tgtd"
)

type Env interface {
    Logger() logger.Logger
    GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
    TelegramAccountRunWithAPI(ctx context.Context, account db.TelegramAccount, f tgtd.APIFunc) error
    TelegramClient() *tgtd.Client  // For GetChannelMessagesWithAPI
    LatestNoteViews() *appmodel.NoteViews
    // WithTransaction executes function within a database transaction
    WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
    // PushNotesTx saves a single note within the current transaction
    // Must be called inside WithTransaction callback
    PushNotesTx(ctx context.Context, note appmodel.RawNote) error
    // InsertNoteAssetStreaming saves asset from io.Reader to avoid OOM on large files
    InsertNoteAssetStreaming(ctx context.Context, path string, reader io.Reader) error
    PrepareLatestNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

type Result struct {
    ImportedCount int
    SkippedCount  int
    AssetsCount   int
    Errors        []string
}

// messageInfo stores pre-computed info for two-pass processing
type messageInfo struct {
    msg      *tg.Message
    title    string
    filename string
    skip     bool      // true if already imported
    hasMedia bool      // true if message has downloadable media
}

func Resolve(ctx context.Context, env Env, params appmodel.ImportTelegramChannelParams) (*Result, error) {
    log := logger.WithPrefix(env.Logger(), "importtelegramchannel:")

    result := &Result{
        Errors: []string{},
    }

    // Get telegram account
    account, err := env.GetTelegramAccountByID(ctx, params.AccountID)
    if err != nil {
        return nil, fmt.Errorf("failed to get telegram account: %w", err)
    }

    // Build map of existing imported notes
    nvs := env.LatestNoteViews()
    importedNotes := appmodel.BuildImportedNotesMap(nvs)
    log.Info("loaded existing notes", "count", len(importedNotes))

    // Collect all messages and download media using a SINGLE connection
    // This is critical to avoid FloodWait and account bans!
    // Media is streamed directly to storage to avoid OOM on large files
    var allMessages []*tg.Message
    downloadedMedia := make(map[int][]tgtd.DownloadedMedia)  // messageID -> metadata only

    assetsDir := fmt.Sprintf("%s/assets", params.BasePath)

    err = env.TelegramAccountRunWithAPI(ctx, account, func(ctx context.Context, api *tg.Client) error {
        tgClient := env.TelegramClient()
        offsetID := 0

        // PHASE 1: Fetch all messages
        for {
            msgResult, fetchErr := tgClient.GetChannelMessagesWithAPI(ctx, api, tgtd.GetChannelMessagesParams{
                ChannelID: params.ChannelID,
                Limit:     100,
                OffsetID:  offsetID,
            })
            if fetchErr != nil {
                return fmt.Errorf("failed to fetch messages: %w", fetchErr)
            }

            if len(msgResult.Messages) == 0 {
                break
            }

            allMessages = append(allMessages, msgResult.Messages...)
            offsetID = msgResult.Messages[len(msgResult.Messages)-1].ID

            log.Info("fetched messages batch", "count", len(msgResult.Messages), "total", len(allMessages))

            if !msgResult.HasMore {
                break
            }
        }

        log.Info("total messages fetched", "count", len(allMessages))

        // PHASE 2: Download and save all media within the same connection
        // Uses STREAMING to avoid loading large files (1.5GB+ videos) into memory
        for _, msg := range allMessages {
            if msg.Media == nil {
                continue
            }

            // Skip if already imported
            key := fmt.Sprintf("%d:%d", params.ChannelID, msg.ID)
            if _, exists := importedNotes[key]; exists {
                continue
            }

            // Stream download directly to storage via callback
            onMedia := func(filename string, mimeType string, reader io.Reader) error {
                assetPath := fmt.Sprintf("%s/%s", assetsDir, filename)
                return env.InsertNoteAssetStreaming(ctx, assetPath, reader)
            }

            media, downloadErr := tgtd.DownloadMessageMediaStreaming(ctx, api, msg, onMedia)
            if downloadErr != nil {
                log.Warn("failed to download media", "msgID", msg.ID, "error", downloadErr)
                continue
            }

            if len(media) > 0 {
                downloadedMedia[msg.ID] = media  // Store metadata only, data is already saved
                result.AssetsCount += len(media)
                log.Info("downloaded and saved media", "msgID", msg.ID, "count", len(media))
            }
        }

        log.Info("media download complete", "messagesWithMedia", len(downloadedMedia))

        return nil
    })

    if err != nil {
        return nil, fmt.Errorf("telegram API error: %w", err)
    }

    // ========================================
    // PASS 1: Generate titles and build postMap
    // ========================================
    // This allows wikilinks to work in BOTH directions (forward and backward references)

    usedFilenames := make(map[string]bool)
    postMap := make(map[string]string)       // messageID -> title (for wikilinks)
    messageInfos := make([]messageInfo, len(allMessages))

    // Process oldest first (reverse order)
    for i := len(allMessages) - 1; i >= 0; i-- {
        msg := allMessages[i]
        idx := len(allMessages) - 1 - i  // Index in messageInfos

        // Check if already imported
        key := fmt.Sprintf("%d:%d", params.ChannelID, msg.ID)
        if _, exists := importedNotes[key]; exists {
            messageInfos[idx] = messageInfo{msg: msg, skip: true}
            result.SkippedCount++
            continue
        }

        // Convert and extract title
        markdown := tgtd.Convert(msg)
        title := extractTitle(markdown)
        if title == "" {
            title = fmt.Sprintf("message-%d", msg.ID)
        }

        // Generate unique filename
        filename := generateFilename(title, msg.ID, usedFilenames)
        usedFilenames[filename] = true

        // Store for wikilink resolution (full map built before Pass 2)
        titleWithoutExt := strings.TrimSuffix(filename, ".md")
        postMap[fmt.Sprintf("%d", msg.ID)] = titleWithoutExt

        // Check if message has downloadable media
        hasMedia := messageHasMedia(msg)

        messageInfos[idx] = messageInfo{
            msg:      msg,
            title:    titleWithoutExt,
            filename: filename,
            skip:     false,
            hasMedia: hasMedia,
        }
    }

    log.Info("pass 1 complete", "postMapSize", len(postMap), "toImport", len(postMap))

    // ========================================
    // PASS 2: Create notes with full postMap
    // ========================================
    // Now all titles are known, so wikilinks can resolve to any post (past or future)
    // Notes are inserted ONE BY ONE within a transaction for consistency
    // If server crashes mid-import, already saved notes persist, re-run will skip them

    err = env.WithTransaction(ctx, func(txCtx context.Context) error {
        for _, info := range messageInfos {
            if info.skip {
                continue
            }

            // Convert message to markdown
            markdown := tgtd.Convert(info.msg)

            // Replace telegram links with wikilinks (using COMPLETE postMap)
            markdown = replaceTelegramLinks(markdown, postMap)

            // Build asset links from metadata (assets already saved during streaming phase)
            var assetLinks []string
            if mediaList, hasMedia := downloadedMedia[info.msg.ID]; hasMedia {
                for _, media := range mediaList {
                    // Build relative link from note to asset
                    // Note is at {basePath}/{filename}.md, asset is at {basePath}/assets/{filename}
                    relativeAssetPath := fmt.Sprintf("assets/%s", media.Filename)

                    // Determine if it's an image or video
                    if strings.HasPrefix(media.MimeType, "image/") {
                        assetLinks = append(assetLinks, fmt.Sprintf("![%s](%s)", media.Filename, relativeAssetPath))
                    } else if strings.HasPrefix(media.MimeType, "video/") {
                        assetLinks = append(assetLinks, fmt.Sprintf("[Video: %s](%s)", media.Filename, relativeAssetPath))
                    }
                }
            }

            // Prepend asset links to markdown if any
            if len(assetLinks) > 0 {
                assetSection := strings.Join(assetLinks, "\n\n") + "\n\n"
                markdown = assetSection + markdown
            }

            // Generate frontmatter
            frontmatter := generateFrontmatter(params.ChannelID, info.msg)

            // Full note content
            content := frontmatter + markdown

            // Full path
            path := fmt.Sprintf("%s/%s", params.BasePath, info.filename)

            // Insert note within transaction (one by one)
            note := appmodel.RawNote{
                Path:    path,
                Content: content,
            }

            insertErr := env.PushNotesTx(txCtx, note)
            if insertErr != nil {
                errMsg := fmt.Sprintf("failed to insert note %s: %v", path, insertErr)
                result.Errors = append(result.Errors, errMsg)
                log.Warn(errMsg)
                // Continue with other notes, don't fail entire transaction
                continue
            }

            result.ImportedCount++
            log.Info("imported message", "id", info.msg.ID, "path", path)
        }
        return nil
    })

    if err != nil {
        return nil, fmt.Errorf("transaction failed: %w", err)
    }

    // Refresh notes after import
    _, err = env.PrepareLatestNotes(ctx)
    if err != nil {
        log.Warn("failed to prepare latest notes after import", "error", err)
    }

    log.Info("import completed",
        "imported", result.ImportedCount,
        "skipped", result.SkippedCount,
        "assets", result.AssetsCount,
        "errors", len(result.Errors))

    return result, nil
}

// messageHasMedia checks if a message has downloadable media (photo/video)
func messageHasMedia(msg *tg.Message) bool {
    if msg.Media == nil {
        return false
    }

    switch msg.Media.(type) {
    case *tg.MessageMediaPhoto:
        return true
    case *tg.MessageMediaDocument:
        return true
    default:
        return false
    }
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
```

> **Two-Pass Processing**: Pass 1 builds the complete `postMap` (messageID -> title) for ALL messages. Pass 2 creates notes using the full map, so wikilinks work in both directions - a post can reference a future post, and the link will resolve correctly.

**File**: `internal/case/backjob/importtelegramchannel/title.go`

Port title extraction logic from `cmd/channelexport/step1_titles.go`:

```go
package importtelegramchannel

import (
    "fmt"
    "regexp"
    "strings"
)

var (
    customEmojiRegex         = regexp.MustCompile(`!\[[^\]]*\]\((tg://emoji\?id=\d+|https://ce\.trip2g\.com/\d+\.webp)\)`)
    malformedEmojiRegex      = regexp.MustCompile(`!\[[^\]]*\]\(tg://emoji\?id=\d+\)>[^<]*</u>`)
    numberedEmojiPrefixRegex = regexp.MustCompile(`^!\[[^\]]*\]\([^)]+\)[\.\s]*`)
    markdownLinkRegex        = regexp.MustCompile(`\[([^\]]*)\]\([^)]+\)`)
    htmlTagRegex             = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
    timecodeRegex            = regexp.MustCompile(`\d{1,2}:\d{2}(?::\d{2})?\s*`)
    leadingJunkRegex         = regexp.MustCompile(`^[\x{1F300}-\x{1F9FF}\x{1F3FB}-\x{1F3FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{25A0}-\x{25FF}\x{2B00}-\x{2BFF}\x{FE00}-\x{FE0F}\x{200D}\s\-–—•·°№#@!?\.,;:\*"'«»„"'']+`)
)

func extractTitle(content string) string {
    text := content

    // Remove malformed custom emoji first
    text = malformedEmojiRegex.ReplaceAllString(text, "")

    // Remove custom emoji markdown
    text = customEmojiRegex.ReplaceAllString(text, "")

    // Remove HTML tags
    text = htmlTagRegex.ReplaceAllString(text, "")

    // Convert markdown links to just text
    text = markdownLinkRegex.ReplaceAllString(text, "$1")

    // Remove markdown formatting
    text = strings.ReplaceAll(text, "**", "")
    text = strings.ReplaceAll(text, "*", "")
    text = strings.ReplaceAll(text, "__", "")
    text = strings.ReplaceAll(text, "_", "")
    text = strings.ReplaceAll(text, "`", "")

    // Remove timecodes
    text = timecodeRegex.ReplaceAllString(text, "")

    // Get first non-empty line
    var firstParagraph string
    for _, line := range strings.Split(text, "\n") {
        line = strings.TrimSpace(line)
        if line != "" {
            firstParagraph = line
            break
        }
    }

    // Remove numbered emoji prefix
    firstParagraph = numberedEmojiPrefixRegex.ReplaceAllString(firstParagraph, "")

    // Strip leading junk repeatedly
    for {
        cleaned := leadingJunkRegex.ReplaceAllString(firstParagraph, "")
        cleaned = strings.TrimSpace(cleaned)
        if cleaned == firstParagraph {
            break
        }
        firstParagraph = cleaned
    }

    // Take first 7 words
    words := strings.Fields(firstParagraph)
    if len(words) > 7 {
        words = words[:7]
    }

    title := strings.Join(words, " ")

    // Remove invalid filename characters
    invalidChars := []string{"/", "\\", ":", "?", "\"", "<", ">", "|", "[", "]", "(", ")", "#"}
    for _, char := range invalidChars {
        title = strings.ReplaceAll(title, char, "")
    }

    // Strip trailing punctuation
    title = strings.TrimRight(title, ".,;:!?…-–—")

    return strings.TrimSpace(title)
}

func generateFilename(title string, messageID int, usedFilenames map[string]bool) string {
    baseFilename := title + ".md"

    if !usedFilenames[baseFilename] {
        return baseFilename
    }

    return fmt.Sprintf("%s (%d).md", title, messageID)
}
```

**File**: `internal/case/backjob/importtelegramchannel/title_test.go`

Unit tests for title extraction:

```go
package importtelegramchannel

import "testing"

func TestExtractTitle(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "simple text",
            input:    "Hello world this is a test message",
            expected: "Hello world this is a test",
        },
        {
            name:     "more than 7 words truncates",
            input:    "one two three four five six seven eight nine ten",
            expected: "one two three four five six seven",
        },
        {
            name:     "with emoji prefix strips emoji",
            input:    "🔥 Breaking news about something important",
            expected: "Breaking news about something important",
        },
        {
            name:     "with custom emoji markdown",
            input:    "![emoji](tg://emoji?id=123) Title here today",
            expected: "Title here today",
        },
        {
            name:     "with markdown links extracts text",
            input:    "[Click here](https://example.com) for more info",
            expected: "Click here for more info",
        },
        {
            name:     "with timecodes removes them",
            input:    "00:15 Introduction to the topic today",
            expected: "Introduction to the topic today",
        },
        {
            name:     "empty after cleaning returns empty",
            input:    "🔥🔥🔥",
            expected: "",
        },
        {
            name:     "invalid filename chars removed",
            input:    "What is this? A test: yes/no",
            expected: "What is this A test yesno",
        },
        {
            name:     "trailing punctuation stripped",
            input:    "This is a title...",
            expected: "This is a title",
        },
        {
            name:     "bold markdown removed",
            input:    "**Bold title** with text here",
            expected: "Bold title with text here",
        },
        {
            name:     "multiline takes first paragraph",
            input:    "First line title\n\nSecond paragraph content",
            expected: "First line title",
        },
        {
            name:     "html tags removed",
            input:    "<b>Bold</b> and <i>italic</i> text",
            expected: "Bold and italic text",
        },
        {
            name:     "numbered emoji prefix stripped",
            input:    "![1](tg://emoji?id=123). First item here",
            expected: "First item here",
        },
        {
            name:     "ce.trip2g.com emoji stripped",
            input:    "![emoji](https://ce.trip2g.com/123.webp) Content here",
            expected: "Content here",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := extractTitle(tt.input)
            if got != tt.expected {
                t.Errorf("extractTitle() = %q, want %q", got, tt.expected)
            }
        })
    }
}

func TestGenerateFilename(t *testing.T) {
    tests := []struct {
        name          string
        title         string
        messageID     int
        usedFilenames map[string]bool
        expected      string
    }{
        {
            name:          "unique title",
            title:         "My Title",
            messageID:     123,
            usedFilenames: map[string]bool{},
            expected:      "My Title.md",
        },
        {
            name:          "duplicate title adds message ID",
            title:         "My Title",
            messageID:     456,
            usedFilenames: map[string]bool{"My Title.md": true},
            expected:      "My Title (456).md",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := generateFilename(tt.title, tt.messageID, tt.usedFilenames)
            if got != tt.expected {
                t.Errorf("generateFilename() = %q, want %q", got, tt.expected)
            }
        })
    }
}
```

**File**: `internal/case/backjob/importtelegramchannel/wikilinks.go`

Port wikilink replacement logic from `cmd/channelexport/step2_wikilinks.go`:

```go
package importtelegramchannel

import (
    "fmt"
    "regexp"
)

var (
    // Matches [text](https://t.me/channel/123) - any channel
    tgLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(https?://t\.me/[^/]+/(\d+)\)`)
    // Custom emoji with tg://emoji?id=123
    customEmojiReplaceRegex = regexp.MustCompile(`!\[([^\]]*)\]\(tg://emoji\?id=(\d+)\)`)
)

func replaceTelegramLinks(content string, postMap map[string]string) string {
    // Replace telegram channel links with wikilinks
    result := tgLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
        submatches := tgLinkRegex.FindStringSubmatch(match)
        if len(submatches) < 3 {
            return match
        }

        postID := submatches[2]

        // Look up in map
        if title, ok := postMap[postID]; ok {
            return fmt.Sprintf("[[%s]]", title)
        }

        // Not found - keep original link
        return match
    })

    // Replace custom emoji tg://emoji?id=... with https://ce.trip2g.com/{id}.webp
    result = customEmojiReplaceRegex.ReplaceAllStringFunc(result, func(match string) string {
        submatches := customEmojiReplaceRegex.FindStringSubmatch(match)
        if len(submatches) < 3 {
            return match
        }
        altText := submatches[1]
        emojiID := submatches[2]
        return fmt.Sprintf("![%s](https://ce.trip2g.com/%s.webp)", altText, emojiID)
    })

    return result
}
```

### 6. Add Server Methods

**File**: `cmd/server/telegram.go`

Add methods for running with API and job enqueue:

```go
// TelegramClient returns the tgtd.Client instance for use with RunWithAPI
func (a *app) TelegramClient() *tgtd.Client {
    // You might cache this or create per-account
    return a.telegramClient  // or create new if needed
}

// TelegramAccountRunWithAPI runs a function with an active Telegram connection.
// All Telegram API calls (fetching messages, downloading media) should happen
// inside this callback to use a SINGLE connection and avoid FloodWait.
func (a *app) TelegramAccountRunWithAPI(ctx context.Context, account db.TelegramAccount, f tgtd.APIFunc) error {
    client := tgtd.NewClient(int(account.ApiID), account.ApiHash)
    return client.RunWithAPI(ctx, account.SessionData, f)
}

// EnqueueImportTelegramChannel enqueues import job to telegramTaskQueue (NOT telegramAPIQueue!)
func (a *app) EnqueueImportTelegramChannel(ctx context.Context, params model.ImportTelegramChannelParams) (string, error) {
    // IMPORTANT: Use telegramTaskQueue, not telegramAPIQueue!
    // telegramAPIQueue has limit=1 and is for bot messages - import would block all notifications
    return a.enqueueTask(ctx, a.telegramTaskQueue, "import_telegram_channel", params)
}
```

**File**: `cmd/server/main.go` (or appropriate location)

Add method to save note assets with streaming:

```go
// InsertNoteAssetStreaming saves an asset file using streaming to avoid OOM.
// Uses io.Reader instead of []byte to handle large files (1.5GB+ videos).
// path format: {basePath}/assets/{filename}
func (a *app) InsertNoteAssetStreaming(ctx context.Context, path string, reader io.Reader) error {
    // Option 1: Stream directly to MinIO/S3
    // This is the recommended approach - data goes directly to storage
    // without loading into server memory
    //
    // Example using MinIO client:
    // _, err := a.minioClient.PutObject(ctx, bucket, path, reader, -1, minio.PutObjectOptions{})
    // return err

    // Option 2: Stream to file system (for git-based storage)
    // Create parent directories if needed, then stream to file
    //
    // if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
    //     return err
    // }
    // file, err := os.Create(fullPath)
    // if err != nil {
    //     return err
    // }
    // defer file.Close()
    // _, err = io.Copy(file, reader)
    // return err

    // For consistency with existing asset handling, consider using
    // the same approach as internal/case/uploadnoteasset/resolve.go
    // which uses PutAssetObject (should be modified to accept io.Reader)

    return a.gitapi.InsertAssetStreaming(path, reader)
}
```

> **CRITICAL**: The `InsertNoteAssetStreaming` must use streaming (io.Reader) instead of loading the entire file into memory. A 1.5GB video would crash the server with OOM if loaded into []byte.
>
> Review `internal/gitapi/api.go` and `internal/case/uploadnoteasset/resolve.go` for existing patterns, but ensure they support streaming.

### 7. Register Background Job

**File**: `cmd/server/queue.go` (or appropriate job registration file)

Register the background job handler:

```go
import "trip2g/internal/case/backjob/importtelegramchannel"

// In job handler registration:
case "import_telegram_channel":
    var params model.ImportTelegramChannelParams
    if err := json.Unmarshal(job.Params, &params); err != nil {
        return fmt.Errorf("failed to unmarshal params: %w", err)
    }
    _, err := importtelegramchannel.Resolve(ctx, a, params)
    return err
```

### 8. Wire Up GraphQL

**File**: `internal/graph/resolver.go`

Add case Env to main Env interface:
```go
import "trip2g/internal/case/admin/importtelegramaccountchannel"

type Env interface {
    // ...existing methods...
    importtelegramaccountchannel.Env
}
```

**File**: `internal/graph/schema.resolvers.go`

Add resolver:
```go
func (r *adminMutationResolver) ImportTelegramAccountChannel(
    ctx context.Context,
    obj *appmodel.AdminMutation,
    input model.AdminImportTelegramAccountChannelInput,
) (model.AdminImportTelegramAccountChannelOrErrorPayload, error) {
    return importtelegramaccountchannel.Resolve(ctx, r.env(ctx), input)
}
```

## Implementation Sequence

1. Add to `internal/tgtd/client.go`:
   - `RunWithAPI` method (single connection wrapper)
   - `GetChannelMessagesWithAPI` method (uses existing connection)
   - `DownloadMessageMediaStreaming` function (streams to storage via callback)
   - `MediaDownloadFunc` type for streaming callback
2. Add duplicate detection methods to `internal/model/note_telegram.go`
3. Add `ImportTelegramChannelParams` to `internal/model/telegram.go`
4. Create `internal/case/admin/importtelegramaccountchannel/resolve.go` (Layer 1)
5. Create `internal/case/backjob/importtelegramchannel/` package (Layer 2):
   - `resolve.go` - main logic with transaction, single connection, streaming, two-pass processing
   - `title.go` - title extraction
   - `title_test.go` - unit tests for title extraction
   - `wikilinks.go` - link replacement
6. Add GraphQL schema types to `internal/graph/schema.graphqls`
7. Run `make gqlgen`
8. Add server methods to `cmd/server/telegram.go`:
   - `TelegramClient()` - returns client for API operations
   - `TelegramAccountRunWithAPI()` - runs callback with single connection
   - `EnqueueImportTelegramChannel()` - enqueues background job
9. Add server methods to `cmd/server/case_methods.go`:
   - `WithTransaction()` - wraps function in database transaction
   - `PushNotesTx()` - saves single note within transaction
10. Add `InsertNoteAssetStreaming(path, io.Reader)` method (MUST use streaming!)
11. Register background job in `cmd/server/queue.go`
12. Update `internal/graph/resolver.go` with new Env interface
13. Implement GraphQL resolver in `internal/graph/schema.resolvers.go`
14. Run `make lint` and `go test ./...`

## Key Considerations

### Single Connection Pattern (CRITICAL)
- **DO NOT create a new connection for each API call!**
- All Telegram API operations MUST happen inside `RunWithAPI` callback
- Creating new connections for each message/media download will cause:
  - FloodWait errors from Telegram
  - Potential account ban
  - Very slow performance (handshake overhead)
- The import uses a single connection for:
  1. Fetching all messages (with pagination)
  2. Downloading all media
- Only after the connection is closed do we process and save notes

### Queue Selection (CRITICAL)
- **DO NOT use `telegramAPIQueue` (`tg_api_jobs`)** - has limit=1 and is for bot notifications
- **USE `telegramTaskQueue` (`tg_task_jobs`)** - for long-running user account operations
- Import can take 10-30 minutes; using wrong queue would block all notifications

### Pagination Bug Fix
- Use `rawCount >= limit` instead of `len(msgList) == limit`
- `rawCount` is the count BEFORE filtering empty messages
- Without this fix, pagination stops early when batch contains media-only messages

### Two-Pass Processing
- **Pass 1**: Generate all titles and build complete `postMap` (messageID -> title)
- **Pass 2**: Create notes with full `postMap` available
- This allows wikilinks to resolve in BOTH directions (forward and backward references)

### Path Sanitization
- Layer 1 sanitizes `basePath` before enqueueing
- Prevents path traversal attacks (`..`)
- Cleans path and removes leading slash

### Transaction Handling
- Notes are inserted **one by one** within `WithTransaction`
- Uses `PushNotesTx(ctx, note)` instead of batch insert
- If server crashes mid-import:
  - Already committed notes are preserved
  - Re-running import skips existing notes (via `LatestNoteViews()` check)
  - Only remaining notes are imported
- Individual note errors don't fail the entire transaction (logged and skipped)

### Context & Timeouts
- Background job runs with `context.Background()` (via goqite)
- Not tied to HTTP request context timeout
- Large imports can run for extended periods

### Error Handling
- Background job continues on individual message errors
- Collects errors in result for logging
- Logs warnings for skipped messages

### Duplicate Detection & Re-import Behavior
- Checks `telegram_publish_channel_id` AND `telegram_publish_message_id` in `LatestNoteViews()`
- Both must match to consider as duplicate (skip download)
- Map built from `LatestNoteViews()` before import starts
- **Deleted files are re-imported**: If user deletes a note file, it won't appear in `LatestNoteViews()`, so re-running import will download it again
- **Incremental import**: Only new posts (not in `LatestNoteViews()`) are fetched and saved
- **Idempotent**: Running import multiple times is safe — existing notes are skipped

### Filename Collisions
- Tracks used filenames during import
- Appends message ID if title collision occurs
- Sanitizes filenames (removes path traversal chars)

### Asset Downloading & Streaming (CRITICAL)
- **MUST use streaming** to avoid OOM on large files (videos can be 1.5GB+)
- `DownloadMessageMediaStreaming` streams directly from Telegram to storage
- `InsertNoteAssetStreaming` accepts `io.Reader`, not `[]byte`
- Uses `io.Pipe()` to connect downloader to storage writer
- Data never fully loaded into memory - streamed in chunks
- Assets saved to `{basePath}/assets/` directory
- Filenames format: `{messageID}_{photoID}.jpg` or `{messageID}_{docID}.ext`
- Downloads largest available photo size
- Supports photos and video documents
- Asset links prepended to note content as markdown images/links
- Errors during asset download are logged but don't fail note import

### Media Groups
- Media groups (multiple photos/videos) are handled per-message
- Each message in a media group has its own media attachment
- All media from a message is downloaded and linked

### Storage Strategy
- Assets are stored via `InsertNoteAsset` method
- Implementation should match existing asset handling pattern
- Can use MinIO/S3 (like `UploadNoteAsset`) or git-based storage
- Review `internal/gitapi/api.go` for git-based approach

### Markdown Formatting Edge Cases

#### Punctuation at Bold Boundaries

CommonMark has strict rules for emphasis delimiters. Opening `**` must be a "left-flanking delimiter run":
- NOT followed by whitespace
- Either (a) NOT followed by punctuation, OR (b) preceded by whitespace/punctuation

**Problem**: Telegram bold entities may include leading/trailing punctuation:
```
Telegram bold: ", но ментально вернулся к жизни только сегодня"
Naive output:  вернулся в субботу**, но ... сегодня**
```

The `**,` sequence doesn't open bold because `**` is followed by punctuation (`,`) and preceded by a word character.

**Solution**: `trimEntitySpaces` in `internal/tgtd/convert.go` trims:
- Leading/trailing spaces (already implemented)
- Leading/trailing punctuation (TODO: implement)

**Expected output**:
```
вернулся в субботу, **но ментально вернулся к жизни только сегодня**
```

The comma moves outside the bold markers.

**Affected punctuation**: `,`, `.`, `!`, `?`, `;`, `:` and similar.

**Note**: 100% perfect import is not achievable. Some edge cases will require manual fixes.
