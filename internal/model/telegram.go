package model

type TelegramPost struct {
	MessageID  *int64 `json:"message_id,omitempty"`
	NotePathID int64  `json:"note_path_id"`

	DBChatID       int64 `json:"chat_id"`
	TelegramChatID int64 `json:"telegram_chat_id"`

	Media   []string `json:"media"`
	Content string   `json:"content"`

	Warnings []string `json:"warnings"`

	LinkCount           int64 `json:"link_count"`
	UnresolvedLinkCount int64 `json:"unresolved_link_count"`
	ExternalLinkCount   int64 `json:"external_link_count"`
}

type TelegramPostSource struct {
	NoteView *NoteView
	ChatID   int64
	Instant  bool
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

type CustomEmojiSticker struct {
	ID         string
	Base64Data string
}
