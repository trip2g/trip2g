package main

import (
	"fmt"
	"sort"
	"strings"
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
// Uses event-based approach: open/close markers at entity boundaries
func applyEntities(text string, entities []tg.MessageEntityClass) string {
	if len(entities) == 0 {
		return text
	}

	// Collect entity information
	infos := make([]entityInfo, 0, len(entities))
	for _, e := range entities {
		info := extractEntityInfo(e)
		if info != nil {
			infos = append(infos, *info)
		}
	}

	if len(infos) == 0 {
		return text
	}

	// Convert UTF-16 offsets to UTF-8
	for i := range infos {
		startUTF8 := utf16ToUTF8Offset(text, infos[i].offset)
		endUTF8 := utf16ToUTF8Offset(text, infos[i].offset+infos[i].length)
		infos[i].offset = startUTF8
		infos[i].length = endUTF8 // reuse as end position
	}

	// Create events for entity starts and ends
	type event struct {
		pos      int
		isEnd    bool
		idx      int
		startAt  int // entity start position
		endAt    int // entity end position
		priority int // lower = outer (opens first, closes last)
	}
	events := make([]event, 0, len(infos)*2)

	for i, info := range infos {
		priority := entityPriority(info.entityType)
		events = append(events,
			event{pos: info.offset, isEnd: false, idx: i, startAt: info.offset, endAt: info.length, priority: priority},
			event{pos: info.length, isEnd: true, idx: i, startAt: info.offset, endAt: info.length, priority: priority},
		)
	}

	// Sort events: by position, ends before starts at same pos, proper nesting order
	sort.Slice(events, func(i, j int) bool {
		if events[i].pos != events[j].pos {
			return events[i].pos < events[j].pos
		}
		// At same position: ends before starts
		if events[i].isEnd != events[j].isEnd {
			return events[i].isEnd
		}
		// Both ends at same position
		if events[i].isEnd {
			// Later started closes first (LIFO)
			if events[i].startAt != events[j].startAt {
				return events[i].startAt > events[j].startAt
			}
			// Same start - higher priority (inner) closes first
			if events[i].priority != events[j].priority {
				return events[i].priority > events[j].priority
			}
			return events[i].idx > events[j].idx
		}
		// Both starts at same position: longer entity opens first (outer before inner)
		if events[i].endAt != events[j].endAt {
			return events[i].endAt > events[j].endAt
		}
		// Same length - lower priority (outer) opens first
		if events[i].priority != events[j].priority {
			return events[i].priority < events[j].priority
		}
		return events[i].idx < events[j].idx
	})

	// Build result with open/close markers
	var result strings.Builder
	prevPos := 0
	activeStack := make([]int, 0) // stack of active entity indices (for reopening after \n\n)

	for _, ev := range events {
		// Output text before this event, handling paragraph breaks
		if ev.pos > prevPos && ev.pos <= len(text) {
			segment := text[prevPos:ev.pos]
			writeSegmentWithParagraphBreaks(&result, segment, infos, activeStack)
			prevPos = ev.pos
		}

		info := &infos[ev.idx]
		if ev.isEnd {
			result.WriteString(closeMarker(info))
			// Remove from active stack
			for i := len(activeStack) - 1; i >= 0; i-- {
				if activeStack[i] == ev.idx {
					activeStack = append(activeStack[:i], activeStack[i+1:]...)
					break
				}
			}
		} else {
			result.WriteString(openMarker(info))
			activeStack = append(activeStack, ev.idx)
		}
	}

	// Output remaining text
	if prevPos < len(text) {
		result.WriteString(text[prevPos:])
	}

	return result.String()
}

// writeSegmentWithParagraphBreaks writes text, closing/reopening formatting at \n\n boundaries
func writeSegmentWithParagraphBreaks(result *strings.Builder, text string, infos []entityInfo, active []int) {
	if len(active) == 0 {
		result.WriteString(text)
		return
	}

	parts := strings.Split(text, "\n\n")
	for i, part := range parts {
		result.WriteString(part)
		if i < len(parts)-1 {
			// Close all active formatting before paragraph break
			for j := len(active) - 1; j >= 0; j-- {
				result.WriteString(closeMarker(&infos[active[j]]))
			}
			result.WriteString("\n\n")
			// Only reopen if there's actual content remaining
			hasContent := false
			for k := i + 1; k < len(parts); k++ {
				if strings.TrimSpace(parts[k]) != "" {
					hasContent = true
					break
				}
			}
			if hasContent {
				for j := 0; j < len(active); j++ {
					result.WriteString(openMarker(&infos[active[j]]))
				}
			}
		}
	}
}

func openMarker(info *entityInfo) string {
	switch info.entityType {
	case "bold":
		return "**"
	case "italic":
		return "*"
	case "code":
		return "`"
	case "pre":
		if info.language != "" {
			return "```" + info.language + "\n"
		}
		return "```\n"
	case "text_link":
		return "["
	case "url":
		return "["
	case "strikethrough":
		return "~~"
	case "underline":
		return "<u>"
	case "spoiler":
		return "||"
	case "custom_emoji":
		return "!["
	default:
		return ""
	}
}

// entityPriority returns priority for nesting order (lower = outer wrapper)
func entityPriority(entityType string) int {
	switch entityType {
	case "text_link", "url":
		return 0 // links are outermost
	case "pre":
		return 1 // code blocks
	case "code":
		return 2 // inline code
	case "spoiler":
		return 3
	case "underline":
		return 4
	case "strikethrough":
		return 5
	case "bold":
		return 6
	case "italic":
		return 7
	case "custom_emoji":
		return 10 // emoji is innermost - bold/italic should wrap it
	default:
		return 20
	}
}

func closeMarker(info *entityInfo) string {
	switch info.entityType {
	case "bold":
		return "**"
	case "italic":
		return "*"
	case "code":
		return "`"
	case "pre":
		return "\n```"
	case "text_link":
		return "](" + info.url + ")"
	case "url":
		return "](" + info.url + ")"
	case "strikethrough":
		return "~~"
	case "underline":
		return "</u>"
	case "spoiler":
		return "||"
	case "custom_emoji":
		return fmt.Sprintf("](https://ce.trip2g.com/%d.webp)", info.documentID)
	default:
		return ""
	}
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
	case *tg.MessageEntityBlockquote:
		// Blockquote - skip for now, markdown doesn't have good equivalent
		return nil
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
