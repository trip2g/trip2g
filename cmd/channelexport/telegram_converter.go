package main

import (
	"fmt"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/gotd/td/tg"
)

// ConvertToMarkdown converts a Telegram message to Markdown format
func ConvertToMarkdown(msg *tg.Message, channelID int64) (string, map[string]interface{}) {
	text := msg.Message
	entities := msg.Entities

	// Convert text with entities to markdown
	markdown := applyEntities(text, entities)

	// Create frontmatter
	frontmatter := map[string]interface{}{
		"telegram_publish_channel_id": fmt.Sprintf("%d", channelID),
		"telegram_publish_message_id": msg.ID,
		"telegram_publish_at":         time.Unix(int64(msg.Date), 0).Format(time.RFC3339),
	}

	return markdown, frontmatter
}

// applyEntities applies Telegram entities to text and converts to Markdown
func applyEntities(text string, entities []tg.MessageEntityClass) string {
	if len(entities) == 0 {
		return text
	}

	// Collect entity information
	sortedEntities := make([]entityInfo, 0, len(entities))
	for _, e := range entities {
		info := extractEntityInfo(e)
		if info != nil {
			sortedEntities = append(sortedEntities, *info)
		}
	}

	// Sort by offset descending (process from end to beginning)
	sort.Slice(sortedEntities, func(i, j int) bool {
		return sortedEntities[i].offset > sortedEntities[j].offset
	})

	// Apply entities in reverse order
	result := text
	for _, info := range sortedEntities {
		result = applySingleEntity(result, info)
	}

	return result
}

type entityInfo struct {
	entityType string
	offset     int
	length     int
	url        string
	language   string
	documentID int64
}

func extractEntityInfo(e tg.MessageEntityClass) *entityInfo {
	switch entity := e.(type) {
	case *tg.MessageEntityBold:
		return &entityInfo{entityType: "bold", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityItalic:
		return &entityInfo{entityType: "italic", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityCode:
		return &entityInfo{entityType: "code", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityPre:
		return &entityInfo{entityType: "pre", offset: entity.Offset, length: entity.Length, language: entity.Language}
	case *tg.MessageEntityTextURL:
		return &entityInfo{entityType: "text_link", offset: entity.Offset, length: entity.Length, url: entity.URL}
	case *tg.MessageEntityURL:
		return &entityInfo{entityType: "url", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityMention:
		return &entityInfo{entityType: "mention", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityHashtag:
		return &entityInfo{entityType: "hashtag", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityStrike:
		return &entityInfo{entityType: "strikethrough", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityUnderline:
		return &entityInfo{entityType: "underline", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntitySpoiler:
		return &entityInfo{entityType: "spoiler", offset: entity.Offset, length: entity.Length}
	case *tg.MessageEntityCustomEmoji:
		return &entityInfo{entityType: "custom_emoji", offset: entity.Offset, length: entity.Length, documentID: entity.DocumentID}
	default:
		return nil
	}
}

func applySingleEntity(text string, entity entityInfo) string {
	// Extract the portion of text this entity applies to
	start := utf16ToUTF8Offset(text, entity.offset)
	end := utf16ToUTF8Offset(text, entity.offset+entity.length)

	if start < 0 || end > len(text) || start >= end {
		return text
	}

	before := text[:start]
	content := text[start:end]
	after := text[end:]

	var wrapped string
	switch entity.entityType {
	case "bold":
		wrapped = fmt.Sprintf("**%s**", content)
	case "italic":
		wrapped = fmt.Sprintf("*%s*", content)
	case "code":
		wrapped = fmt.Sprintf("`%s`", content)
	case "pre":
		// Code block
		language := ""
		if entity.language != "" {
			language = entity.language
		}
		wrapped = fmt.Sprintf("```%s\n%s\n```", language, content)
	case "text_link":
		wrapped = fmt.Sprintf("[%s](%s)", content, entity.url)
	case "url":
		// Already a URL, wrap in markdown link format
		wrapped = fmt.Sprintf("[%s](%s)", content, content)
	case "mention":
		// @username - keep as is
		wrapped = content
	case "hashtag":
		// #hashtag - keep as is
		wrapped = content
	case "strikethrough":
		wrapped = fmt.Sprintf("~~%s~~", content)
	case "underline":
		// Markdown doesn't have native underline, use HTML
		wrapped = fmt.Sprintf("<u>%s</u>", content)
	case "spoiler":
		// Use spoiler syntax (some markdown flavors support ||text||)
		wrapped = fmt.Sprintf("||%s||", content)
	case "custom_emoji":
		// Custom emoji as markdown image with tg:// URL
		wrapped = fmt.Sprintf("![%s](tg://emoji?id=%d)", content, entity.documentID)
	default:
		// Unknown entity type, keep as is
		wrapped = content
	}

	return before + wrapped + after
}

// utf16ToUTF8Offset converts UTF-16 offset to UTF-8 offset
func utf16ToUTF8Offset(text string, utf16Offset int) int {
	if utf16Offset == 0 {
		return 0
	}

	utf8Offset := 0
	utf16Count := 0

	for utf8Offset < len(text) {
		if utf16Count >= utf16Offset {
			break
		}

		r, size := utf8.DecodeRuneInString(text[utf8Offset:])
		utf8Offset += size

		// Count UTF-16 code units for this rune
		if r <= 0xFFFF {
			utf16Count++
		} else {
			// Surrogate pair
			utf16Count += 2
		}
	}

	return utf8Offset
}
