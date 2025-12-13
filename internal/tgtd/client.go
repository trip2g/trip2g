package tgtd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/valyala/fasthttp"

	"trip2g/internal/logger"
)

const (
	maxFloodWaitRetries = 3

	// Video file extensions.
	extMP4  = ".mp4"
	extAVI  = ".avi"
	extMOV  = ".mov"
	extMKV  = ".mkv"
	extWEBM = ".webm"
	extM4V  = ".m4v"
)

// ClientEnv provides dependencies for Client.
type ClientEnv interface {
	Logger() logger.Logger
	DecryptData(ciphertext []byte) ([]byte, error)
}

// retryOnFloodWait executes fn and retries on FLOOD_WAIT errors.
func retryOnFloodWait[T any](ctx context.Context, log logger.Logger, fn func() (T, error)) (T, error) {
	var zero T
	for i := range maxFloodWaitRetries {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		waited, waitErr := tgerr.FloodWait(ctx, err)
		if !waited {
			// Not a flood wait error or context cancelled
			return zero, waitErr
		}

		// Log and retry
		if d, ok := tgerr.AsFloodWait(err); ok {
			log.Info("FLOOD_WAIT: retrying after delay", "delay", d, "attempt", i+1)
		}
	}

	return zero, errors.New("max retries exceeded for FLOOD_WAIT")
}

// Client wraps the gotd/td Telegram client for user account operations.
type Client struct {
	apiID   int
	apiHash string
	log     logger.Logger
}

// NewClient creates a new Client.
func NewClient(env ClientEnv, apiID int, apiHash string) *Client {
	return &Client{
		apiID:   apiID,
		apiHash: apiHash,
		log:     logger.WithPrefix(env.Logger(), "tgtd:client:"),
	}
}

// safeCtx creates a safe context for Telegram operations.
//
// WHY THIS IS NEEDED:
// When using fasthttpadaptor to convert fasthttp requests to net/http (for GraphQL),
// the fasthttp.RequestCtx is passed directly as context.Context to the net/http handler.
// gotd/td uses net/http client which creates persistent HTTP connections with goroutines.
// These goroutines may outlive the fasthttp request. When fasthttp finishes the request,
// it calls RequestCtx.Reset() which causes a DATA RACE:
//   - fasthttp goroutine: writes to RequestCtx.userData (in Reset())
//   - gotd HTTP goroutine: reads from RequestCtx.userData (for cancellation)
//
// SOLUTION:
// Detect fasthttp.RequestCtx and replace it with an independent context.Background()
// with timeout, so Telegram HTTP connections don't hold references to the recycled RequestCtx.
func safeCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	// Check if this is a fasthttp context
	if _, ok := ctx.(*fasthttp.RequestCtx); ok {
		// Create independent context with timeout to avoid data race
		return context.WithTimeout(context.Background(), 5*time.Minute)
	}

	// For non-fasthttp contexts, caller is responsible for timeouts
	return ctx, func() {}
}

// ChatInfo contains information about a Telegram chat.
type ChatInfo struct {
	ID       int64
	Title    string
	ChatType string // "channel", "group", "supergroup", "private"
}

// DialogInfo contains information about a Telegram dialog (user, channel, or group).

// DialogType represents the type of a Telegram dialog.
type DialogType string

const (
	DialogTypeUser    DialogType = "user"
	DialogTypeChannel DialogType = "channel"
	DialogTypeChat    DialogType = "chat"
)

type DialogInfo struct {
	ID       int64
	Username string
	Title    string
	Type     DialogType
}

// ListChats returns all chats/channels the account has access to.
func (c *Client) ListChats(ctx context.Context, sessionData []byte) ([]ChatInfo, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var chats []ChatInfo

	err = client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get dialogs (chats)
		dialogs, getDialogsErr := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      100,
		})
		if getDialogsErr != nil {
			return fmt.Errorf("failed to get dialogs: %w", getDialogsErr)
		}

		var chatList []tg.ChatClass
		switch d := dialogs.(type) {
		case *tg.MessagesDialogs:
			chatList = d.Chats
		case *tg.MessagesDialogsSlice:
			chatList = d.Chats
		}

		for _, chat := range chatList {
			switch ch := chat.(type) {
			case *tg.Channel:
				chatType := "channel"
				if ch.Megagroup {
					chatType = "supergroup"
				}
				chats = append(chats, ChatInfo{
					ID:       ch.ID,
					Title:    ch.Title,
					ChatType: chatType,
				})
			case *tg.Chat:
				chats = append(chats, ChatInfo{
					ID:       ch.ID,
					Title:    ch.Title,
					ChatType: "group",
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return chats, nil
}

// ListDialogs returns all dialogs (users, channels, groups) the account has.
func (c *Client) ListDialogs(ctx context.Context, sessionData []byte) ([]DialogInfo, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var dialogs []DialogInfo

	err = client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get dialogs
		dialogsResp, getDialogsErr := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      100,
		})
		if getDialogsErr != nil {
			return fmt.Errorf("failed to get dialogs: %w", getDialogsErr)
		}

		var chatList []tg.ChatClass
		var userList []tg.UserClass
		switch d := dialogsResp.(type) {
		case *tg.MessagesDialogs:
			chatList = d.Chats
			userList = d.Users
		case *tg.MessagesDialogsSlice:
			chatList = d.Chats
			userList = d.Users
		}

		// Add users
		for _, user := range userList {
			if u, ok := user.(*tg.User); ok {
				if u.Bot || u.Self {
					continue
				}
				title := u.FirstName
				if u.LastName != "" {
					title += " " + u.LastName
				}
				dialogs = append(dialogs, DialogInfo{
					ID:       u.ID,
					Username: u.Username,
					Title:    title,
					Type:     DialogTypeUser,
				})
			}
		}

		// Add channels and groups
		for _, chat := range chatList {
			switch ch := chat.(type) {
			case *tg.Channel:
				dialogs = append(dialogs, DialogInfo{
					ID:       ch.ID,
					Username: ch.Username,
					Title:    ch.Title,
					Type:     DialogTypeChannel,
				})
			case *tg.Chat:
				dialogs = append(dialogs, DialogInfo{
					ID:       ch.ID,
					Username: "",
					Title:    ch.Title,
					Type:     DialogTypeChat,
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return dialogs, nil
}

// SendMessageParams contains parameters for sending a message.
type SendMessageParams struct {
	ChatID    int64
	Message   string
	NoWebpage bool
}

// SendPhotoParams contains parameters for sending a photo.
type SendPhotoParams struct {
	ChatID   int64
	PhotoURL string
	Caption  string
}

// SendMediaGroupParams contains parameters for sending a media group.
type SendMediaGroupParams struct {
	ChatID    int64
	MediaURLs []string
	Caption   string
}

// SendMessageResult contains the result of sending a message.
type SendMessageResult struct {
	MessageID int64
}

// SendMessage sends a text message to a chat with HTML formatting.
func (c *Client) SendMessage(ctx context.Context, sessionData []byte, params SendMessageParams) (*SendMessageResult, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var result *SendMessageResult

	err = client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get peer for the chat
		peer, peerErr := c.resolvePeer(ctx, api, params.ChatID)
		if peerErr != nil {
			return fmt.Errorf("failed to resolve peer: %w", peerErr)
		}

		// Parse HTML to get text and entities
		eb := entity.Builder{}
		if parseErr := html.HTML(strings.NewReader(params.Message), &eb, html.Options{}); parseErr != nil {
			return fmt.Errorf("failed to parse HTML: %w", parseErr)
		}
		messageText, entities := eb.Complete()

		// Debug custom emoji entities
		for _, ent := range entities {
			if ce, ok := ent.(*tg.MessageEntityCustomEmoji); ok {
				c.log.Debug("SendMessage: CustomEmoji entity",
					"offset", ce.Offset,
					"length", ce.Length,
					"document_id", ce.DocumentID,
				)
			}
		}

		// Build request with optional NoWebpage flag
		req := &tg.MessagesSendMessageRequest{
			Peer:      peer,
			Message:   messageText,
			Entities:  entities,
			NoWebpage: params.NoWebpage,
			RandomID:  rand.Int64(), //nolint:gosec // G404: RandomID is for message deduplication, not security
		}

		// Send message using raw API
		updates, sendErr := api.MessagesSendMessage(ctx, req)
		if sendErr != nil {
			return fmt.Errorf("failed to send message: %w", sendErr)
		}

		// Extract message ID from updates
		messageID, extractErr := extractMessageID(updates)
		if extractErr != nil {
			return fmt.Errorf("failed to extract message ID: %w", extractErr)
		}

		result = &SendMessageResult{
			MessageID: messageID,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// SendPhoto sends a photo to a chat with HTML formatted caption.
func (c *Client) SendPhoto(ctx context.Context, sessionData []byte, params SendPhotoParams) (*SendMessageResult, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var result *SendMessageResult

	err = client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get peer for the chat
		peer, peerErr := c.resolvePeer(ctx, api, params.ChatID)
		if peerErr != nil {
			return fmt.Errorf("failed to resolve peer: %w", peerErr)
		}

		// Download photo from URL
		photoData, downloadErr := downloadMedia(ctx, params.PhotoURL)
		if downloadErr != nil {
			return fmt.Errorf("failed to download photo: %w", downloadErr)
		}

		// Upload photo using uploader
		up := uploader.NewUploader(api)
		fileName := filenameFromURL(params.PhotoURL)
		uploaded, uploadErr := up.FromBytes(ctx, fileName, photoData)
		if uploadErr != nil {
			return fmt.Errorf("failed to upload photo: %w", uploadErr)
		}

		// Use message sender with HTML formatting for caption
		sender := message.NewSender(api)

		updates, sendErr := sender.To(peer).UploadedPhoto(ctx, uploaded, html.String(nil, params.Caption))
		if sendErr != nil {
			return fmt.Errorf("failed to send photo: %w", sendErr)
		}

		// Extract message ID from updates
		messageID, extractErr := extractMessageID(updates)
		if extractErr != nil {
			return fmt.Errorf("failed to extract message ID: %w", extractErr)
		}

		result = &SendMessageResult{
			MessageID: messageID,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// SendMediaGroup sends a media group (multiple photos/videos) to a chat.
func (c *Client) SendMediaGroup(ctx context.Context, sessionData []byte, params SendMediaGroupParams) (*SendMessageResult, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var result *SendMessageResult

	err = client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get peer for the chat
		peer, peerErr := c.resolvePeer(ctx, api, params.ChatID)
		if peerErr != nil {
			return fmt.Errorf("failed to resolve peer: %w", peerErr)
		}

		// Upload all media files
		up := uploader.NewUploader(api)
		mediaInputs, uploadErr := uploadMediaFiles(ctx, up, params.MediaURLs, params.Caption)
		if uploadErr != nil {
			return uploadErr
		}

		// Use message sender to send media group
		sender := message.NewSender(api)

		updates, sendErr := sender.To(peer).Album(ctx, mediaInputs[0], mediaInputs[1:]...)
		if sendErr != nil {
			return fmt.Errorf("failed to send media group: %w", sendErr)
		}

		// Extract message ID from updates
		messageID, extractErr := extractMessageID(updates)
		if extractErr != nil {
			return fmt.Errorf("failed to extract message ID: %w", extractErr)
		}

		result = &SendMessageResult{
			MessageID: messageID,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// uploadMediaFiles downloads and uploads multiple media files, returning MultiMediaOptions.
func uploadMediaFiles(ctx context.Context, up *uploader.Uploader, mediaURLs []string, caption string) ([]message.MultiMediaOption, error) {
	var mediaInputs []message.MultiMediaOption

	for i, mediaURL := range mediaURLs {
		// Download media from URL
		mediaData, downloadErr := downloadMedia(ctx, mediaURL)
		if downloadErr != nil {
			return nil, fmt.Errorf("failed to download media %s: %w", mediaURL, downloadErr)
		}

		// Upload media
		fileName := filenameFromURL(mediaURL)
		uploaded, uploadErr := up.FromBytes(ctx, fileName, mediaData)
		if uploadErr != nil {
			return nil, fmt.Errorf("failed to upload media %s: %w", mediaURL, uploadErr)
		}

		// Only first media item gets caption
		includeCaption := i == 0 && caption != ""
		mediaOpt := createMultiMediaOption(uploaded, mediaURL, caption, includeCaption)

		mediaInputs = append(mediaInputs, mediaOpt)
	}

	return mediaInputs, nil
}

// createMultiMediaOption creates a MultiMediaOption for photo or video based on URL extension.
func createMultiMediaOption(uploaded tg.InputFileClass, mediaURL, caption string, includeCaption bool) message.MultiMediaOption {
	fileName := filenameFromURL(mediaURL)
	ext := strings.ToLower(filepath.Ext(fileName))
	isVideo := isVideoExtension(ext)

	if isVideo {
		mimeType := videoMIMEType(ext)
		videoAttr := &tg.DocumentAttributeVideo{
			SupportsStreaming: true,
			W:                 1280,
			H:                 720,
			Duration:          1,
		}
		videoAttr.SetFlags()

		if includeCaption {
			return message.UploadedDocument(uploaded, html.String(nil, caption)).
				MIME(mimeType).
				Attributes(
					videoAttr,
					&tg.DocumentAttributeFilename{
						FileName: fileName,
					},
				)
		}
		return message.UploadedDocument(uploaded).
			MIME(mimeType).
			Attributes(
				videoAttr,
				&tg.DocumentAttributeFilename{
					FileName: fileName,
				},
			)
	}

	if includeCaption {
		return message.UploadedPhoto(uploaded, html.String(nil, caption))
	}
	return message.UploadedPhoto(uploaded)
}

// downloadMedia downloads media from URL and returns the bytes.
func downloadMedia(ctx context.Context, mediaURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// isVideoExtension returns true if the extension is a video format.
func isVideoExtension(ext string) bool {
	switch ext {
	case extMP4, extAVI, extMOV, extMKV, extWEBM, extM4V:
		return true
	default:
		return false
	}
}

// videoMIMEType returns the MIME type for a video extension.
func videoMIMEType(ext string) string {
	switch ext {
	case extMP4, extM4V:
		return "video/mp4"
	case extAVI:
		return "video/x-msvideo"
	case extMOV:
		return "video/quicktime"
	case extMKV:
		return "video/x-matroska"
	case extWEBM:
		return "video/webm"
	default:
		return "video/mp4"
	}
}

// filenameFromURL extracts a clean filename from a URL, removing query parameters.
// If no valid extension is found, defaults to .jpg for images.
func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return filepath.Base(rawURL)
	}

	// Get just the path without query parameters
	baseName := filepath.Base(parsed.Path)

	// Check if it has a valid extension
	ext := strings.ToLower(filepath.Ext(baseName))
	if ext == "" {
		// No extension, default to .jpg
		return baseName + ".jpg"
	}

	return baseName
}

// EditMessageParams contains parameters for editing a message.
type EditMessageParams struct {
	ChatID    int64
	MessageID int64
	Message   string
}

// EditMessage edits an existing message with HTML formatting.
func (c *Client) EditMessage(ctx context.Context, sessionData []byte, params EditMessageParams) error {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get peer for the chat
		peer, peerErr := c.resolvePeer(ctx, api, params.ChatID)
		if peerErr != nil {
			return fmt.Errorf("failed to resolve peer: %w", peerErr)
		}

		// Use message sender with HTML formatting for edit
		sender := message.NewSender(api)

		_, editErr := sender.To(peer).Edit(int(params.MessageID)).StyledText(ctx, html.String(nil, params.Message))
		if editErr != nil {
			return fmt.Errorf("failed to edit message: %w", editErr)
		}

		return nil
	})
}

// EditMessageWithPhotoParams contains parameters for editing a message to add a photo.
type EditMessageWithPhotoParams struct {
	ChatID    int64
	MessageID int64
	PhotoURL  string
	Caption   string
}

// EditMessageWithPhoto edits an existing message to add a photo with HTML formatted caption.
func (c *Client) EditMessageWithPhoto(ctx context.Context, sessionData []byte, params EditMessageWithPhotoParams) error {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get peer for the chat
		peer, peerErr := c.resolvePeer(ctx, api, params.ChatID)
		if peerErr != nil {
			return fmt.Errorf("failed to resolve peer: %w", peerErr)
		}

		// Download photo from URL
		photoData, downloadErr := downloadMedia(ctx, params.PhotoURL)
		if downloadErr != nil {
			return fmt.Errorf("failed to download photo: %w", downloadErr)
		}

		// Upload photo using uploader
		up := uploader.NewUploader(api)
		fileName := filenameFromURL(params.PhotoURL)
		uploaded, uploadErr := up.FromBytes(ctx, fileName, photoData)
		if uploadErr != nil {
			return fmt.Errorf("failed to upload photo: %w", uploadErr)
		}

		// Parse caption with HTML
		eb := entity.Builder{}
		if parseErr := html.HTML(strings.NewReader(params.Caption), &eb, html.Options{}); parseErr != nil {
			return fmt.Errorf("failed to parse caption HTML: %w", parseErr)
		}
		captionText, entities := eb.Complete()

		// Edit message with photo
		_, editErr := api.MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
			Peer:     peer,
			ID:       int(params.MessageID),
			Message:  captionText,
			Entities: entities,
			Media: &tg.InputMediaUploadedPhoto{
				File: uploaded,
			},
		})
		if editErr != nil {
			return fmt.Errorf("failed to edit message with photo: %w", editErr)
		}

		return nil
	})
}

// EditMessageCaptionParams contains parameters for editing a message caption.
type EditMessageCaptionParams struct {
	ChatID    int64
	MessageID int64
	Caption   string
}

// EditMessageCaption edits the caption of a photo or media group message with HTML formatting.
func (c *Client) EditMessageCaption(ctx context.Context, sessionData []byte, params EditMessageCaptionParams) error {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get peer for the chat
		peer, peerErr := c.resolvePeer(ctx, api, params.ChatID)
		if peerErr != nil {
			return fmt.Errorf("failed to resolve peer: %w", peerErr)
		}

		// Parse caption with HTML
		eb := entity.Builder{}
		if parseErr := html.HTML(strings.NewReader(params.Caption), &eb, html.Options{}); parseErr != nil {
			return fmt.Errorf("failed to parse caption HTML: %w", parseErr)
		}
		captionText, entities := eb.Complete()

		// Edit message caption (without changing media)
		_, editErr := api.MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
			Peer:     peer,
			ID:       int(params.MessageID),
			Message:  captionText,
			Entities: entities,
		})
		if editErr != nil {
			return fmt.Errorf("failed to edit message caption: %w", editErr)
		}

		return nil
	})
}

// DeleteMessageParams contains parameters for deleting a message.
type DeleteMessageParams struct {
	ChatID    int64
	MessageID int64
}

// DeleteMessage deletes a message from a chat.
func (c *Client) DeleteMessage(ctx context.Context, sessionData []byte, params DeleteMessageParams) error {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Get peer for the chat
		peer, peerErr := c.resolvePeer(ctx, api, params.ChatID)
		if peerErr != nil {
			return fmt.Errorf("failed to resolve peer: %w", peerErr)
		}

		// Delete message using the appropriate method based on peer type
		switch p := peer.(type) {
		case *tg.InputPeerChannel:
			_, delErr := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
				Channel: &tg.InputChannel{
					ChannelID:  p.ChannelID,
					AccessHash: p.AccessHash,
				},
				ID: []int{int(params.MessageID)},
			})
			if delErr != nil {
				return fmt.Errorf("failed to delete channel message: %w", delErr)
			}
		default:
			_, delErr := api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
				ID: []int{int(params.MessageID)},
			})
			if delErr != nil {
				return fmt.Errorf("failed to delete message: %w", delErr)
			}
		}

		return nil
	})
}

func (c *Client) resolvePeer(ctx context.Context, api *tg.Client, chatID int64) (tg.InputPeerClass, error) {
	// Try to get channel/chat from dialogs
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	var chatList []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chatList = d.Chats
	case *tg.MessagesDialogsSlice:
		chatList = d.Chats
	}

	for _, chat := range chatList {
		switch ch := chat.(type) {
		case *tg.Channel:
			if ch.ID == chatID {
				return &tg.InputPeerChannel{
					ChannelID:  ch.ID,
					AccessHash: ch.AccessHash,
				}, nil
			}
		case *tg.Chat:
			if ch.ID == chatID {
				return &tg.InputPeerChat{
					ChatID: ch.ID,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("chat with ID %d not found in dialogs", chatID)
}

func extractMessageID(updates tg.UpdatesClass) (int64, error) {
	switch u := updates.(type) {
	case *tg.Updates:
		for _, update := range u.Updates {
			if msgUpdate, ok := update.(*tg.UpdateMessageID); ok {
				return int64(msgUpdate.ID), nil
			}
		}
		// If no UpdateMessageID, try to find from UpdateNewMessage
		for _, update := range u.Updates {
			if newMsg, newMsgOk := update.(*tg.UpdateNewMessage); newMsgOk {
				if msg, msgOk := newMsg.Message.(*tg.Message); msgOk {
					return int64(msg.ID), nil
				}
			}
			if newMsg, chanMsgOk := update.(*tg.UpdateNewChannelMessage); chanMsgOk {
				if msg, msgOk := newMsg.Message.(*tg.Message); msgOk {
					return int64(msg.ID), nil
				}
			}
		}
	case *tg.UpdateShortSentMessage:
		return int64(u.ID), nil
	}

	return 0, errors.New("could not extract message ID from updates")
}

// AppConfig contains the result of help.getAppConfig.
type AppConfig struct {
	JSON string
}

// GetAppConfig fetches the app configuration from Telegram.
func (c *Client) GetAppConfig(ctx context.Context, sessionData []byte) (*AppConfig, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var result *AppConfig

	err = client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Call help.getAppConfig with hash=0 to get full config
		appConfig, configErr := api.HelpGetAppConfig(ctx, 0)
		if configErr != nil {
			return fmt.Errorf("failed to get app config: %w", configErr)
		}

		switch cfg := appConfig.(type) {
		case *tg.HelpAppConfig:
			// Convert JSONValue to JSON string
			jsonStr, jsonErr := jsonValueToString(cfg.Config)
			if jsonErr != nil {
				return fmt.Errorf("failed to convert config to JSON: %w", jsonErr)
			}
			result = &AppConfig{
				JSON: jsonStr,
			}
		case *tg.HelpAppConfigNotModified:
			// This shouldn't happen with hash=0, but handle it
			result = &AppConfig{
				JSON: "{}",
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// jsonValueToString converts a tg.JSONValueClass to a JSON string.
func jsonValueToString(value tg.JSONValueClass) (string, error) {
	result, err := jsonValueToInterface(value)
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// jsonValueToInterface converts a tg.JSONValueClass to a Go interface{}.
func jsonValueToInterface(value tg.JSONValueClass) (interface{}, error) {
	switch v := value.(type) {
	case *tg.JSONNull:
		return nil, nil
	case *tg.JSONBool:
		return v.Value, nil
	case *tg.JSONNumber:
		return v.Value, nil
	case *tg.JSONString:
		return v.Value, nil
	case *tg.JSONArray:
		arr := make([]interface{}, len(v.Value))
		for i, item := range v.Value {
			converted, err := jsonValueToInterface(item)
			if err != nil {
				return nil, err
			}
			arr[i] = converted
		}
		return arr, nil
	case *tg.JSONObject:
		obj := make(map[string]interface{})
		for _, item := range v.Value {
			converted, err := jsonValueToInterface(item.Value)
			if err != nil {
				return nil, err
			}
			obj[item.Key] = converted
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("unknown JSON value type: %T", value)
	}
}

// UserInfo contains information about the authenticated user.
type UserInfo struct {
	IsPremium bool
}

// GetUserInfo fetches information about the authenticated user.
func (c *Client) GetUserInfo(ctx context.Context, sessionData []byte) (*UserInfo, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

	storage := &session.StorageMemory{}
	err := storage.StoreSession(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: storage,
	})

	var result *UserInfo

	err = client.Run(ctx, func(ctx context.Context) error {
		self, selfErr := client.Self(ctx)
		if selfErr != nil {
			return fmt.Errorf("failed to get self: %w", selfErr)
		}

		result = &UserInfo{
			IsPremium: self.Premium,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// APIFunc is a function that receives the Telegram API client.
type APIFunc func(ctx context.Context, api *tg.Client) error

// RunWithAPI runs a function with an active Telegram API connection.
// Use this to perform multiple operations within a single session.
// This avoids creating new connections for each operation, preventing FloodWait.
func (c *Client) RunWithAPI(ctx context.Context, sessionData []byte, f APIFunc) error {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

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

// GetChannelMessagesParams contains parameters for fetching channel messages.
type GetChannelMessagesParams struct {
	ChannelID int64
	Limit     int // Max messages to fetch per batch (max 100)
	OffsetID  int // Message ID to start from (for pagination, 0 = from latest)
}

// GetChannelMessagesResult contains the result of fetching channel messages.
type GetChannelMessagesResult struct {
	Messages []*tg.Message
	HasMore  bool
}

// GetChannelMessagesWithAPI fetches messages using an existing API connection.
// IMPORTANT: Use this within RunWithAPI callback, not standalone!
//
//nolint:gocognit // complex message processing with multiple type assertions
func (c *Client) GetChannelMessagesWithAPI(ctx context.Context, api *tg.Client, params GetChannelMessagesParams) (*GetChannelMessagesResult, error) {
	ctx, cancel := safeCtx(ctx)
	defer cancel()

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

	// Get history with retry on FLOOD_WAIT
	messages, err := retryOnFloodWait(ctx, c.log, func() (tg.MessagesMessagesClass, error) {
		return api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     channelPeer,
			Limit:    limit,
			OffsetID: params.OffsetID,
		})
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
		HasMore:  rawCount >= limit, // Use rawCount, not filtered len!
	}, nil
}

func (c *Client) resolvePeerWithAPI(ctx context.Context, api *tg.Client, chatID int64) (tg.InputPeerClass, error) {
	// Try to get channel/chat from dialogs with retry on FLOOD_WAIT
	dialogs, err := retryOnFloodWait(ctx, c.log, func() (tg.MessagesDialogsClass, error) {
		return api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      100,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	var chatList []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chatList = d.Chats
	case *tg.MessagesDialogsSlice:
		chatList = d.Chats
	}

	for _, chat := range chatList {
		switch ch := chat.(type) {
		case *tg.Channel:
			if ch.ID == chatID {
				return &tg.InputPeerChannel{
					ChannelID:  ch.ID,
					AccessHash: ch.AccessHash,
				}, nil
			}
		case *tg.Chat:
			if ch.ID == chatID {
				return &tg.InputPeerChat{
					ChatID: ch.ID,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("chat with ID %d not found in dialogs", chatID)
}

// DownloadedMedia represents downloaded media stored in temp file.
// Caller MUST call Cleanup() when done to remove temp file.
type DownloadedMedia struct {
	Filename   string
	MimeType   string
	Sha256Hash string
	Size       int64
	TempPath   string // path to temp file
	IsImage    bool
	IsVideo    bool
}

// Open returns the downloaded media file.
// Caller must close the file when done.
func (m *DownloadedMedia) Open() (*os.File, error) {
	return os.Open(m.TempPath)
}

// Cleanup removes the temp file.
func (m *DownloadedMedia) Cleanup() {
	if m.TempPath != "" {
		_ = os.Remove(m.TempPath)
	}
}

// MediaInfo represents media metadata without downloading the file.
type MediaInfo struct {
	Filename string
	MimeType string
	IsImage  bool
	IsVideo  bool
}

// GetMessageMediaInfo extracts media metadata from a message without downloading.
// Returns nil if message has no media or media type is not supported.
func GetMessageMediaInfo(msg *tg.Message) []MediaInfo {
	if msg.Media == nil {
		return nil
	}

	var result []MediaInfo

	switch media := msg.Media.(type) {
	case *tg.MessageMediaPhoto:
		if media.Photo == nil {
			return nil
		}
		photo, ok := media.Photo.(*tg.Photo)
		if !ok {
			return nil
		}
		result = append(result, MediaInfo{
			Filename: fmt.Sprintf("%d.jpg", photo.ID),
			MimeType: "image/jpeg",
			IsImage:  true,
		})

	case *tg.MessageMediaDocument:
		if media.Document == nil {
			return nil
		}
		doc, ok := media.Document.(*tg.Document)
		if !ok {
			return nil
		}

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

		if strings.HasPrefix(doc.MimeType, "image/") {
			isImage = true
		}
		if strings.HasPrefix(doc.MimeType, "video/") {
			isVideo = true
		}

		if !isImage && !isVideo {
			return nil
		}

		if filename == "" {
			ext := ".bin"
			if isImage {
				ext = ".jpg"
			} else if isVideo {
				ext = ".mp4"
			}
			filename = fmt.Sprintf("%d%s", doc.ID, ext)
		}

		result = append(result, MediaInfo{
			Filename: filename,
			MimeType: doc.MimeType,
			IsImage:  isImage,
			IsVideo:  isVideo,
		})
	}

	return result
}

// DownloadMessageMedia downloads media from a message to temp files.
// Returns nil if message has no media or media type is not supported.
// IMPORTANT: Use this within RunWithAPI callback, not standalone!
// Caller MUST call Cleanup() on each returned media when done.
//
//nolint:gocognit,funlen // complex media type handling with multiple branches
func DownloadMessageMedia(ctx context.Context, api *tg.Client, msg *tg.Message) ([]DownloadedMedia, error) {
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
		var thumbSize string
		switch s := bestSize.(type) {
		case *tg.PhotoSize:
			thumbSize = s.Type
		case *tg.PhotoSizeProgressive:
			thumbSize = s.Type
		}

		location := &tg.InputPhotoFileLocation{
			ID:            photo.ID,
			AccessHash:    photo.AccessHash,
			FileReference: photo.FileReference,
			ThumbSize:     thumbSize,
		}

		filename := fmt.Sprintf("%d.jpg", photo.ID)
		downloaded, err := downloadToTempFile(ctx, d, api, location, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to download photo: %w", err)
		}
		downloaded.MimeType = "image/jpeg"
		downloaded.IsImage = true
		result = append(result, *downloaded)

	case *tg.MessageMediaDocument:
		if media.Document == nil {
			return nil, nil
		}
		doc, ok := media.Document.(*tg.Document)
		if !ok {
			return nil, nil
		}

		// Check if it's a photo/video
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
			filename = fmt.Sprintf("%d%s", doc.ID, ext)
		}

		// Build input location
		location := &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
		}

		downloaded, err := downloadToTempFile(ctx, d, api, location, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to download document: %w", err)
		}
		downloaded.MimeType = doc.MimeType
		downloaded.IsImage = isImage
		downloaded.IsVideo = isVideo
		result = append(result, *downloaded)
	}

	return result, nil
}

// downloadToTempFile streams from Telegram to temp file, calculating hash.
func downloadToTempFile(
	ctx context.Context,
	d *downloader.Downloader,
	api *tg.Client,
	location tg.InputFileLocationClass,
	filename string,
) (*DownloadedMedia, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "tg-media-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Use TeeReader to calculate hash while downloading
	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	// Download
	_, err = d.Download(api, location).Stream(ctx, writer)
	_ = tmpFile.Close()

	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to download: %w", err)
	}

	// Get file size
	stat, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	return &DownloadedMedia{
		Filename:   filename,
		Sha256Hash: hash,
		Size:       stat.Size(),
		TempPath:   tmpPath,
	}, nil
}
