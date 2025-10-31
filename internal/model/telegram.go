package model

type TelegramPost struct {
	NotePathID int64 `json:"note_path_id"`

	DBChatID       int64 `json:"chat_id"`
	TelegramChatID int64 `json:"telegram_chat_id"`

	Images  []string `json:"images"`
	Content string   `json:"content"`

	Warnings []string `json:"warnings"`

	LinkCount         int64 `json:"link_count"`
	ExternalLinkCount int64 `json:"external_link_count"`
}

type TelegramPostSource struct {
	NoteView *NoteView
	ChatID   int64
	Instant  bool
}
