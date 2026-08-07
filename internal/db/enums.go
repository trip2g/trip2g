package db

// TelegramPublishSentMessage post types.
//
// The column is plain text with no CHECK constraint, so rich needed no
// migration — only a value nothing wrote before.
const (
	TelegramPublishSentMessagePostTypeText       = "text"
	TelegramPublishSentMessagePostTypePhoto      = "photo"
	TelegramPublishSentMessagePostTypeMediaGroup = "media_group"
	TelegramPublishSentMessagePostTypeRich       = "rich"
)

// TelegramPublishSentMessagePostTypeFor classifies a post for storage.
//
// Rich is decided first and media count is never consulted for it: a rich
// post's media rides inside its block tree, and FromMediaCount can only ever
// answer with one of the three classic types. Letting media count decide alone
// is what made a rich row look classic to the update path, which then edited it
// as plain text and destroyed every block.
func TelegramPublishSentMessagePostTypeFor(isRich bool, mediaCount int) string {
	if isRich {
		return TelegramPublishSentMessagePostTypeRich
	}

	return TelegramPublishSentMessagePostTypeFromMediaCount(mediaCount)
}

// TelegramPublishSentMessagePostTypeFromMediaCount returns post type based on media count.
// It cannot return rich; use TelegramPublishSentMessagePostTypeFor when the post
// may be rich.
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
