package model

type TelegramPost struct {
	Images  []string
	Content string

	Warnings []string
}

type TelegramPostSource struct {
	NoteView *NoteView
	ChatID   int64
	Instant  bool
}
