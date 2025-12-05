package model

import (
	"fmt"
	"time"
)

func (note *NoteView) IsTelegramPublishPost() bool {
	_, withPublishAt := note.ExtractTelegramPublishAt(time.UTC)
	_, withPublishTags := note.ExtractTelegramPublishTags()

	return withPublishAt && withPublishTags
}

func (note *NoteView) ExtractTelegramPublishAt(loc *time.Location) (time.Time, bool) {
	rawAt, ok := note.RawMeta["telegram_publish_at"]
	if !ok {
		return time.Time{}, false
	}

	atStr, ok := rawAt.(string)
	if !ok {
		note.AddWarning(NoteWarningWarning, "invalid telegram_publish_at format, expected string")
		return time.Time{}, false
	}

	// parse time with timezone
	at, err := time.Parse(time.RFC3339, atStr)
	if err == nil {
		return at, true
	}

	// parse time without timezone
	at, err = time.ParseInLocation("2006-01-02T15:04:05", atStr, loc)
	if err != nil {
		msg := "failed to parse telegram_publish_at, expected format YYYY-MM-DDTHH:MM:SS (%s)"
		note.AddWarning(NoteWarningWarning, msg, err.Error())
		return time.Time{}, false
	}

	return at, true
}

func (note *NoteView) ExtractTelegramPublishTags() ([]string, bool) {
	rawTags, ok := note.RawMeta["telegram_publish_tags"]
	if !ok {
		return nil, false
	}

	tagsI, ok := rawTags.([]interface{})
	if !ok {
		note.AddWarning(NoteWarningWarning, "invalid telegram_publish_tags format, expected []string")
		return nil, false
	}

	var tags []string

	for _, t := range tagsI {
		tagStr, tagOk := t.(string)
		if !tagOk {
			note.AddWarning(NoteWarningWarning, "invalid tag in telegram_publish_tags, expected string")
			continue
		}

		tags = append(tags, tagStr)
	}

	return tags, true
}

func (note *NoteView) ExtractTelegramPublishDisableWebPagePreview() bool {
	val, ok := note.RawMeta["telegram_publish_disable_web_page_preview"]
	if !ok {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}

	return false
}

// ExtractTelegramPublishChannelID returns the channel ID if present in metadata.
func (note *NoteView) ExtractTelegramPublishChannelID() (int64, bool) {
	rawChannelID, ok := note.RawMeta["telegram_publish_channel_id"]
	if !ok {
		return 0, false
	}
	switch v := rawChannelID.(type) {
	case string:
		id, err := parseInt64(v)
		if err != nil {
			return 0, false
		}
		return id, true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	}
	return 0, false
}

// ExtractTelegramPublishMessageID returns the message ID if present in metadata.
func (note *NoteView) ExtractTelegramPublishMessageID() (int, bool) {
	rawMessageID, ok := note.RawMeta["telegram_publish_message_id"]
	if !ok {
		return 0, false
	}
	switch v := rawMessageID.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}
	return 0, false
}

// BuildImportedNotesMap builds a map of imported notes keyed by "channelID:messageID".
func BuildImportedNotesMap(nvs *NoteViews) map[string]*NoteView {
	result := make(map[string]*NoteView)
	for _, note := range nvs.List {
		channelID, hasChannel := note.ExtractTelegramPublishChannelID()
		messageID, hasMessage := note.ExtractTelegramPublishMessageID()
		if hasChannel && hasMessage {
			key := formatImportKey(channelID, messageID)
			result[key] = note
		}
	}
	return result
}

// FormatImportKey creates a key for imported notes map.
func formatImportKey(channelID int64, messageID int) string {
	return fmt.Sprintf("%d:%d", channelID, messageID)
}

// FormatImportKey creates a key for imported notes map (exported version).
func FormatImportKey(channelID int64, messageID int) string {
	return formatImportKey(channelID, messageID)
}

func parseInt64(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
