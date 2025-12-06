package db

// TelegramPublishSentMessage post types.
const (
	TelegramPublishSentMessagePostTypeText       = "text"
	TelegramPublishSentMessagePostTypePhoto      = "photo"
	TelegramPublishSentMessagePostTypeMediaGroup = "media_group"
)

// TelegramPublishSentMessagePostTypeFromMediaCount returns post type based on media count.
func TelegramPublishSentMessagePostTypeFromMediaCount(mediaCount int) string {
	switch mediaCount {
	case 0:
		return TelegramPublishSentMessagePostTypeText
	case 1:
		return TelegramPublishSentMessagePostTypePhoto
	default:
		return TelegramPublishSentMessagePostTypeMediaGroup
	}
}
