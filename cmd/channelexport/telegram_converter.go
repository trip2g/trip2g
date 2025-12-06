package main

import (
	"fmt"
	"time"

	"github.com/gotd/td/tg"

	"trip2g/internal/tgtd"
)

// ConvertToMarkdown converts a Telegram message to Markdown format.
func ConvertToMarkdown(msg *tg.Message, channelID int64) (string, map[string]interface{}) {
	// Convert text with entities to markdown
	markdown := tgtd.Convert(msg)

	// Create frontmatter
	frontmatter := map[string]interface{}{
		"telegram_publish_channel_id": fmt.Sprintf("%d", channelID),
		"telegram_publish_message_id": msg.ID,
		"telegram_publish_at":         time.Unix(int64(msg.Date), 0).Format(time.RFC3339),
	}

	return markdown, frontmatter
}
