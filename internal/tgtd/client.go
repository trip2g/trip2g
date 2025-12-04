package tgtd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

// Client wraps the gotd/td Telegram client for user account operations.
type Client struct {
	apiID   int
	apiHash string
}

// NewClient creates a new Client.
func NewClient(apiID int, apiHash string) *Client {
	return &Client{
		apiID:   apiID,
		apiHash: apiHash,
	}
}

// ChatInfo contains information about a Telegram chat.
type ChatInfo struct {
	ID       int64
	Title    string
	ChatType string // "channel", "group", "supergroup", "private"
}

// DialogInfo contains information about a Telegram dialog (user, channel, or group).
type DialogInfo struct {
	ID       int64
	Username string
	Title    string
}

// ListChats returns all chats/channels the account has access to.
func (c *Client) ListChats(ctx context.Context, sessionData []byte) ([]ChatInfo, error) {
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
				})
			case *tg.Chat:
				dialogs = append(dialogs, DialogInfo{
					ID:       ch.ID,
					Username: "",
					Title:    ch.Title,
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
				slog.Info("SendMessage: CustomEmoji entity",
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
			RandomID:  rand.Int64(),
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
	case ".mp4", ".avi", ".mov", ".mkv", ".webm", ".m4v":
		return true
	default:
		return false
	}
}

// videoMIMEType returns the MIME type for a video extension.
func videoMIMEType(ext string) string {
	switch ext {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	case ".webm":
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

// DeleteMessageParams contains parameters for deleting a message.
type DeleteMessageParams struct {
	ChatID    int64
	MessageID int64
}

// DeleteMessage deletes a message from a chat.
func (c *Client) DeleteMessage(ctx context.Context, sessionData []byte, params DeleteMessageParams) error {
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
