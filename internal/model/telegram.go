package model

import (
	"strings"
	"trip2g/internal/tgrich"
)

type TelegramPost struct {
	MessageID  *int64 `json:"message_id,omitempty"`
	NotePathID int64  `json:"note_path_id"`

	DBChatID       int64 `json:"chat_id"`
	TelegramChatID int64 `json:"telegram_chat_id"`

	Media   []string `json:"media"`
	Content string   `json:"content"`

	// RichBlocks carries the typed block tree when the note asked for
	// telegram_rich: on against a bot destination. Non-empty selects the
	// sendRichMessage transport; Content is still populated, because the
	// sent-message row stores it and the classic update path still reads it.
	RichBlocks []tgrich.Block `json:"rich_blocks,omitempty"`

	Warnings []string `json:"warnings"`

	LinkCount           int64 `json:"link_count"`
	UnresolvedLinkCount int64 `json:"unresolved_link_count"`
	ExternalLinkCount   int64 `json:"external_link_count"`

	DisableWebPagePreview bool `json:"disable_web_page_preview"`
}

// IsRich reports whether this post is delivered as a rich message.
func (p TelegramPost) IsRich() bool {
	return len(p.RichBlocks) > 0
}

// TelegramPostLink represents a link to a published Telegram post.
// Used by the default template to render "Read in Telegram" links.
type TelegramPostLink struct {
	ChatTitle string // Channel name (from tg_bot_chats.chat_title, may be empty for account messages).
	URL       string // Full t.me URL, e.g. https://t.me/c/1234567890/123
}

// NormalizeTelegramChatID converts Telegram chat ID to channel ID format.
// For channels, removes the -100 prefix (e.g., -1001234567890 -> 1234567890).
// For other chat types, returns the absolute value.
func NormalizeTelegramChatID(chatID int64) int64 {
	if chatID > -1000000000000 && chatID < 0 {
		return -chatID
	}
	// Channel IDs: -100XXXXXXXXXX → strip -100
	if chatID <= -1000000000000 {
		return -(chatID + 1000000000000)
	}
	return chatID
}

// ExtractChannelFromTelegramLink extracts a readable channel identifier from a t.me URL.
// "https://t.me/mychannel/123" -> "mychannel".
// "https://t.me/c/1234567890/123" -> "" (private channel, no readable name).
func ExtractChannelFromTelegramLink(link string) string {
	parts := strings.Split(strings.TrimPrefix(link, "https://t.me/"), "/")
	if len(parts) >= 2 && parts[0] != "c" {
		return parts[0]
	}
	return ""
}

type TelegramPostSource struct {
	NoteView           *NoteView
	ChatID             int64 // Internal DB ID (tg_bot_chats.id) - for bot publishing
	AccountID          int64 // Account ID (telegram_accounts.id) - for account publishing
	TelegramChatID     int64 // Telegram chat ID (e.g., -1001234567890 for channels)
	Instant            bool
	CaptionLengthLimit int
}

// TelegramSendPostParams contains parameters for sending a telegram post.
type TelegramSendPostParams struct {
	NotePathID     int64 `json:"note_path_id"`
	DBChatID       int64 `json:"chat_id"`
	TelegramChatID int64 `json:"telegram_chat_id"`

	Post              TelegramPost `json:"post"`
	Instant           bool         `json:"instant"`
	UpdateLinkedPosts bool         `json:"update_linked_posts"`

	DisableNotification bool `json:"disable_notification"`
}

// TelegramUpdatePostParams contains parameters for updating a telegram post.
type TelegramUpdatePostParams struct {
	TelegramSendPostParams

	MessageID int64 `json:"message_id"`
}

type SendTelegramPublishPostParams struct {
	NotePathID        int64 `json:"note_path_id"`
	Instant           bool  `json:"instant"`
	UpdateLinkedPosts bool  `json:"update_linked_posts"`
}

// TelegramAccountSendPostParams contains parameters for sending a post via user account (MTProto).
type TelegramAccountSendPostParams struct {
	NotePathID     int64 `json:"note_path_id"`
	AccountID      int64 `json:"account_id"`
	TelegramChatID int64 `json:"telegram_chat_id"`

	Post              TelegramPost `json:"post"`
	Instant           bool         `json:"instant"`
	UpdateLinkedPosts bool         `json:"update_linked_posts"`
}

// TelegramAccountUpdatePostParams contains parameters for updating a post via user account.
type TelegramAccountUpdatePostParams struct {
	TelegramAccountSendPostParams

	MessageID int64 `json:"message_id"`
}

// ImportTelegramChannelParams contains parameters for import background job.
type ImportTelegramChannelParams struct {
	AccountID  int64  `json:"account_id"`
	ChannelID  int64  `json:"channel_id"`
	BasePath   string `json:"base_path"`
	WithMedia  bool   `json:"with_media"`  // Download and import media. Default: false
	SkipExists bool   `json:"skip_exists"` // Skip posts that already exist. Default: true
}
