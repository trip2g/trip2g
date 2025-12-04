package model

import "time"

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
