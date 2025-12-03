package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gotd/td/tg"
)

// Entity priorities - lower = outer wrapper (opens first, closes last)
var priorities = map[string]int{
	"mention":               50,
	"hashtag":               50,
	"bot_command":           50,
	"url":                   50,
	"email":                 50,
	"bold":                  90,
	"italic":                91,
	"code":                  20,
	"pre":                   11,
	"text_link":             49,
	"text_mention":          49,
	"underline":             92,
	"strikethrough":         93,
	"blockquote":            0,
	"spoiler":               94,
	"custom_emoji":          99,
	"expandable_blockquote": 0,
}

type entity struct {
	Type       string
	Offset     int // UTF-16 offset
	Length     int // UTF-16 length
	URL        string
	Language   string
	DocumentID int64
}

func getPriority(entityType string) int {
	if p, ok := priorities[entityType]; ok {
		return p
	}
	return 50
}

// ConvertToMarkdownV2 converts tg.Message to markdown using entity-based approach
func ConvertToMarkdownV2(msg *tg.Message) string {
	text := msg.Message
	if len(msg.Entities) == 0 {
		return text
	}

	// Extract entity info
	entities := make([]entity, 0, len(msg.Entities))
	for _, e := range msg.Entities {
		if ent := extractEntity(e); ent != nil {
			entities = append(entities, *ent)
		}
	}

	if len(entities) == 0 {
		return text
	}

	result := applyEntitiesToText(text, entities)

	// Post-process: fix newlines inside links [text\n](url) -> [text](url)\n
	result = fixNewlinesInLinks(result)

	// Post-process: remove redundant adjacent markers (close+open same type)
	result = removeRedundantMarkers(result)

	return result
}

// removeRedundantMarkers removes patterns like **** (close bold + open bold)
func removeRedundantMarkers(text string) string {
	// Only replace adjacent close+open of same type
	// Order matters - replace longer patterns first
	replacements := []struct{ old, new string }{
		{"****", ""},     // close bold + open bold -> nothing
		{"** **", " **"}, // close bold + space + open bold -> space + bold
	}

	for _, r := range replacements {
		text = strings.ReplaceAll(text, r.old, r.new)
	}

	return text
}

// fixNewlinesInLinks moves newlines from inside link text to outside
func fixNewlinesInLinks(text string) string {
	// Pattern: newline followed by ](url)
	// Replace \n]( with ](\n but we need to find the full link first
	var result strings.Builder
	i := 0
	for i < len(text) {
		// Look for pattern: \n](
		if i < len(text)-2 && text[i] == '\n' && text[i+1] == ']' && text[i+2] == '(' {
			// Find the closing )
			j := i + 3
			for j < len(text) && text[j] != ')' {
				j++
			}
			if j < len(text) {
				// Write ](url)\n instead of \n](url)
				result.WriteString(text[i+1 : j+1]) // ](url)
				result.WriteByte('\n')
				i = j + 1
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String()
}

func extractEntity(e tg.MessageEntityClass) *entity {
	switch ent := e.(type) {
	case *tg.MessageEntityBold:
		// Skip single-character bold (often just whitespace)
		if ent.Length <= 1 {
			return nil
		}
		return &entity{Type: "bold", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityItalic:
		return &entity{Type: "italic", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityCode:
		return &entity{Type: "code", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityPre:
		return &entity{Type: "pre", Offset: ent.Offset, Length: ent.Length, Language: ent.Language}
	case *tg.MessageEntityTextURL:
		return &entity{Type: "text_link", Offset: ent.Offset, Length: ent.Length, URL: ent.URL}
	case *tg.MessageEntityURL:
		return &entity{Type: "url", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityStrike:
		return &entity{Type: "strikethrough", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityUnderline:
		return &entity{Type: "underline", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntitySpoiler:
		return &entity{Type: "spoiler", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityCustomEmoji:
		return &entity{Type: "custom_emoji", Offset: ent.Offset, Length: ent.Length, DocumentID: ent.DocumentID}
	case *tg.MessageEntityBlockquote:
		return &entity{Type: "blockquote", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityMention:
		return &entity{Type: "mention", Offset: ent.Offset, Length: ent.Length}
	case *tg.MessageEntityHashtag:
		return &entity{Type: "hashtag", Offset: ent.Offset, Length: ent.Length}
	default:
		return nil
	}
}

type processedEntity struct {
	entity
	startByte int
	endByte   int
}

func applyEntitiesToText(text string, entities []entity) string {
	// Convert UTF-16 offsets to UTF-8 byte positions
	processed := make([]processedEntity, len(entities))
	for i, e := range entities {
		startByte := utf16OffsetToByteOffset(text, e.Offset)
		endByte := utf16OffsetToByteOffset(text, e.Offset+e.Length)
		processed[i] = processedEntity{
			entity:    e,
			startByte: startByte,
			endByte:   endByte,
		}
	}

	// Create events for entity boundaries
	type event struct {
		pos      int
		isEnd    bool
		idx      int
		priority int
		startAt  int
		endAt    int
	}

	events := make([]event, 0, len(processed)*2)
	for i, p := range processed {
		priority := getPriority(p.Type)
		events = append(events,
			event{pos: p.startByte, isEnd: false, idx: i, priority: priority, startAt: p.startByte, endAt: p.endByte},
			event{pos: p.endByte, isEnd: true, idx: i, priority: priority, startAt: p.startByte, endAt: p.endByte},
		)
	}

	// Sort events
	sort.Slice(events, func(i, j int) bool {
		if events[i].pos != events[j].pos {
			return events[i].pos < events[j].pos
		}
		// At same position: ends before starts
		if events[i].isEnd != events[j].isEnd {
			return events[i].isEnd
		}
		// Both ends: later started closes first (LIFO)
		if events[i].isEnd {
			if events[i].startAt != events[j].startAt {
				return events[i].startAt > events[j].startAt
			}
			return events[i].priority > events[j].priority
		}
		// Both starts: longer entity opens first
		if events[i].endAt != events[j].endAt {
			return events[i].endAt > events[j].endAt
		}
		return events[i].priority < events[j].priority
	})

	// Build result
	var result strings.Builder
	prevPos := 0
	activeStack := make([]int, 0)

	pendingNewlines := ""
	for i, ev := range events {
		// Output text segment
		if ev.pos > prevPos && ev.pos <= len(text) {
			segment := text[prevPos:ev.pos]
			// Strip trailing \n\n - will be written after ALL close markers
			if strings.HasSuffix(segment, "\n\n") {
				segment = segment[:len(segment)-2]
				pendingNewlines = "\n\n"
			}
			writeSegment(&result, segment, processed, activeStack)
			prevPos = ev.pos
		}

		p := &processed[ev.idx]
		if ev.isEnd {
			result.WriteString(closeMarkerV2(p))
			// Remove from active stack
			for j := len(activeStack) - 1; j >= 0; j-- {
				if activeStack[j] == ev.idx {
					activeStack = append(activeStack[:j], activeStack[j+1:]...)
					break
				}
			}
			// Write pending newlines after all close markers at this position
			nextIsCloseAtSamePos := i+1 < len(events) && events[i+1].pos == ev.pos && events[i+1].isEnd
			if pendingNewlines != "" && !nextIsCloseAtSamePos {
				result.WriteString(pendingNewlines)
				pendingNewlines = ""
			}
		} else {
			// Write any pending newlines before open marker
			if pendingNewlines != "" {
				result.WriteString(pendingNewlines)
				pendingNewlines = ""
			}
			// Check if we're at a list item - output "- " before open marker
			if ev.pos < len(text) && strings.HasPrefix(text[ev.pos:], "- ") {
				result.WriteString("- ")
				prevPos += 2
			}
			result.WriteString(openMarkerV2(p))
			activeStack = append(activeStack, ev.idx)
		}
	}

	// Output remaining text
	if prevPos < len(text) {
		result.WriteString(text[prevPos:])
	}

	return result.String()
}

func writeSegment(result *strings.Builder, text string, entities []processedEntity, active []int) {
	if len(active) == 0 {
		result.WriteString(text)
		return
	}

	// First split by paragraph breaks
	paragraphs := strings.Split(text, "\n\n")
	for pi, para := range paragraphs {
		// Handle lines within paragraph (for lists)
		lines := strings.Split(para, "\n")
		for i, line := range lines {
			// Check if line is a list item - output "- " outside formatting
			// Skip first line of first paragraph - markers already opened before this function
			trimmed := strings.TrimSpace(line)
			isFirstLine := pi == 0 && i == 0
			if !isFirstLine && strings.HasPrefix(trimmed, "- ") {
				idx := strings.Index(line, "- ")
				prefix := line[:idx+2]
				content := line[idx+2:]

				result.WriteString(prefix)
				for j := 0; j < len(active); j++ {
					result.WriteString(openMarkerV2(&entities[active[j]]))
				}
				result.WriteString(content)
			} else {
				result.WriteString(line)
			}

			// Handle newlines within paragraph
			if i < len(lines)-1 {
				nextLine := lines[i+1]
				isNextListItem := strings.HasPrefix(strings.TrimSpace(nextLine), "- ")

				if isNextListItem {
					// Close all before list item
					for j := len(active) - 1; j >= 0; j-- {
						result.WriteString(closeMarkerV2(&entities[active[j]]))
					}
					result.WriteString("\n")
				} else {
					result.WriteString("\n")
				}
			}
		}

		// Handle paragraph break
		if pi < len(paragraphs)-1 {
			nextPara := paragraphs[pi+1]
			// If next paragraph is empty, this is end of segment - don't close here,
			// let the event close markers handle it
			if strings.TrimSpace(nextPara) == "" {
				result.WriteString("\n\n")
				continue
			}

			// Close all active formatting before paragraph break
			for j := len(active) - 1; j >= 0; j-- {
				result.WriteString(closeMarkerV2(&entities[active[j]]))
			}
			result.WriteString("\n\n")
			// Reopen for next paragraph
			nextTrimmed := strings.TrimSpace(nextPara)
			if strings.HasPrefix(nextTrimmed, "- ") {
				// Don't reopen - list handling will do it
			} else {
				for j := 0; j < len(active); j++ {
					result.WriteString(openMarkerV2(&entities[active[j]]))
				}
			}
		}
	}
}

func openMarkerV2(e *processedEntity) string {
	switch e.Type {
	case "bold":
		return "**"
	case "italic":
		return "*"
	case "code":
		return "`"
	case "pre":
		if e.Language != "" {
			return "```" + e.Language + "\n"
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
	case "blockquote":
		return "> "
	default:
		return ""
	}
}

func closeMarkerV2(e *processedEntity) string {
	switch e.Type {
	case "bold":
		return "**"
	case "italic":
		return "*"
	case "code":
		return "`"
	case "pre":
		return "\n```"
	case "text_link":
		return "](" + e.URL + ")"
	case "url":
		return "](" + e.URL + ")"
	case "strikethrough":
		return "~~"
	case "underline":
		return "</u>"
	case "spoiler":
		return "||"
	case "custom_emoji":
		return fmt.Sprintf("](https://ce.trip2g.com/%d.webp)", e.DocumentID)
	case "blockquote":
		return ""
	default:
		return ""
	}
}

// utf16OffsetToByteOffset converts UTF-16 code unit offset to byte offset
func utf16OffsetToByteOffset(text string, utf16Offset int) int {
	if utf16Offset == 0 {
		return 0
	}

	byteOffset := 0
	utf16Count := 0

	for byteOffset < len(text) && utf16Count < utf16Offset {
		r, size := utf8.DecodeRuneInString(text[byteOffset:])
		byteOffset += size

		// Count UTF-16 code units
		if r <= 0xFFFF {
			utf16Count++
		} else {
			utf16Count += 2 // Surrogate pair
		}
	}

	return byteOffset
}
