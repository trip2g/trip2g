package tgtd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
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

// SendMessageParams contains parameters for sending a message.
type SendMessageParams struct {
	ChatID  int64
	Message string
	// TODO: add media support later
}

// SendMessageResult contains the result of sending a message.
type SendMessageResult struct {
	MessageID int64
}

// SendMessage sends a text message to a chat.
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

		// Send message
		updates, sendErr := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer:     peer,
			Message:  params.Message,
			RandomID: time.Now().UnixNano(),
		})
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

// EditMessageParams contains parameters for editing a message.
type EditMessageParams struct {
	ChatID    int64
	MessageID int64
	Message   string
}

// EditMessage edits an existing message.
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

		// Edit message
		_, editErr := api.MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
			Peer:    peer,
			ID:      int(params.MessageID),
			Message: params.Message,
		})
		if editErr != nil {
			return fmt.Errorf("failed to edit message: %w", editErr)
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
