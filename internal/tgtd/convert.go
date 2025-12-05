package tgtd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"
)

// Format tracks formatting state for a character.
type Format struct {
	Bold          bool
	Italic        bool
	Strikethrough bool
	Underline     bool
	Spoiler       bool
}

func (f Format) Equal(other Format) bool {
	return f.Bold == other.Bold &&
		f.Italic == other.Italic &&
		f.Strikethrough == other.Strikethrough &&
		f.Underline == other.Underline &&
		f.Spoiler == other.Spoiler
}

// Convert converts a Telegram message to Markdown format.
func Convert(msg *tg.Message) string {
	var result string

	// Convert text content
	if len(msg.Message) > 0 {
		result = convertText(msg)
	}

	// Convert poll if present
	if poll, ok := msg.Media.(*tg.MessageMediaPoll); ok {
		pollMarkdown := convertPoll(poll)
		if result != "" {
			result += "\n\n" + pollMarkdown
		} else {
			result = pollMarkdown
		}
	}

	return result
}

// convertPoll converts a Telegram poll to Markdown checkbox list.
func convertPoll(media *tg.MessageMediaPoll) string {
	poll := media.Poll
	results := media.Results

	// Build option -> correct map
	correctOptions := make(map[string]bool)
	for _, r := range results.Results {
		if r.Correct {
			correctOptions[string(r.Option)] = true
		}
	}

	var sb strings.Builder
	sb.WriteString("**")
	sb.WriteString(poll.Question.Text)
	sb.WriteString("**\n\n")

	for _, answer := range poll.Answers {
		optionKey := string(answer.Option)
		if correctOptions[optionKey] {
			sb.WriteString("- [x] ")
		} else {
			sb.WriteString("- [ ] ")
		}
		sb.WriteString(answer.Text.Text)
		sb.WriteString("\n")
	}

	// Remove trailing newline
	result := sb.String()
	return strings.TrimSuffix(result, "\n")
}

// convertText converts message text with entities to Markdown.
func convertText(msg *tg.Message) string {
	source := []rune(msg.Message)
	if len(source) == 0 {
		return ""
	}

	// Preprocess entities: trim leading/trailing spaces from formatting entities
	entities := trimSpacesFromEntities(source, msg.Entities)

	// Build per-character format map
	formats := make([]Format, len(source))
	for _, e := range entities {
		start := utf16OffsetToRune(msg.Message, e.GetOffset())
		end := start + utf16LengthToRune(msg.Message, e.GetOffset(), e.GetLength())

		for i := start; i < end && i < len(formats); i++ {
			switch e.(type) {
			case *tg.MessageEntityBold:
				formats[i].Bold = true
			case *tg.MessageEntityItalic:
				formats[i].Italic = true
			case *tg.MessageEntityStrike:
				formats[i].Strikethrough = true
			case *tg.MessageEntityUnderline:
				formats[i].Underline = true
			case *tg.MessageEntitySpoiler:
				formats[i].Spoiler = true
			}
		}
	}

	// Collect replacing entities (links, emoji, mentions)
	type replacement struct {
		offset int
		length int
		text   string
	}
	var replacements []replacement
	replaced := make([]bool, len(source))

	for _, e := range entities {
		start := utf16OffsetToRune(msg.Message, e.GetOffset())
		length := utf16LengthToRune(msg.Message, e.GetOffset(), e.GetLength())
		text := string(source[start : start+length])

		var replText string
		switch entity := e.(type) {
		case *tg.MessageEntityTextURL:
			linkText := strings.TrimRight(text, "\n")
			trailingNewlines := text[len(linkText):]
			replText = "[" + linkText + "](" + entity.URL + ")" + trailingNewlines
		case *tg.MessageEntityURL:
			// Plain URL - keep as is, markdown will auto-link
			continue
		case *tg.MessageEntityMention:
			// @username - keep as is
			continue
		case *tg.MessageEntityMentionName:
			// Mention with user ID - convert to link
			replText = "[" + text + "](tg://user?id=" + strconv.FormatInt(entity.UserID, 10) + ")"
		case *tg.MessageEntityCustomEmoji:
			replText = fmt.Sprintf("![](https://ce.trip2g.com/%d.webp)", entity.DocumentID)
		case *tg.MessageEntityCode:
			replText = "`" + text + "`"
		case *tg.MessageEntityPre:
			// Code block with optional language
			if entity.Language != "" {
				replText = "```" + entity.Language + "\n" + text + "\n```"
			} else {
				replText = "```\n" + text + "\n```"
			}
		default:
			continue
		}

		for i := start; i < start+length && i < len(replaced); i++ {
			replaced[i] = true
		}
		replacements = append(replacements, replacement{start, length, replText})
	}

	// Render with formatting
	var result strings.Builder
	var currentFmt Format
	atLineStart := true
	replIdx := 0

	for i := 0; i < len(source); i++ {
		r := source[i]

		// Check for replacement
		if replIdx < len(replacements) && replacements[replIdx].offset == i {
			repl := replacements[replIdx]
			// Close formats before replacement
			writeCloseFormats(&result, currentFmt)
			currentFmt = Format{}
			result.WriteString(repl.text)
			i += repl.length - 1
			replIdx++
			atLineStart = strings.HasSuffix(repl.text, "\n")
			continue
		}

		if replaced[i] {
			continue
		}

		// Handle list marker at line start: "- " -> " - "
		if atLineStart && r == '-' && i+1 < len(source) && source[i+1] == ' ' {
			// Close any open formats before list marker
			writeCloseFormats(&result, currentFmt)
			currentFmt = Format{}
			result.WriteString(" - ")
			i++ // skip the space
			atLineStart = false
			continue
		}

		targetFmt := formats[i]

		// Handle newline - close formats before, reopen after if needed
		if r == '\n' {
			writeCloseFormats(&result, currentFmt)
			currentFmt = Format{}
			result.WriteRune(r)
			atLineStart = true
			continue
		}

		atLineStart = false

		// Transition formatting
		if !currentFmt.Equal(targetFmt) {
			// Close formats that are ending (reverse order)
			if currentFmt.Spoiler && !targetFmt.Spoiler {
				result.WriteString("||")
			}
			if currentFmt.Underline && !targetFmt.Underline {
				result.WriteString("</u>")
			}
			if currentFmt.Strikethrough && !targetFmt.Strikethrough {
				result.WriteString("~~")
			}
			if currentFmt.Bold && !targetFmt.Bold {
				result.WriteString("**")
			}
			if currentFmt.Italic && !targetFmt.Italic {
				result.WriteString("*")
			}

			// Open new formats
			if !currentFmt.Italic && targetFmt.Italic {
				result.WriteString("*")
			}
			if !currentFmt.Bold && targetFmt.Bold {
				result.WriteString("**")
			}
			if !currentFmt.Strikethrough && targetFmt.Strikethrough {
				result.WriteString("~~")
			}
			if !currentFmt.Underline && targetFmt.Underline {
				result.WriteString("<u>")
			}
			if !currentFmt.Spoiler && targetFmt.Spoiler {
				result.WriteString("||")
			}

			currentFmt = targetFmt
		}

		result.WriteRune(r)
	}

	// Close any remaining formats
	writeCloseFormats(&result, currentFmt)

	return result.String()
}

// trimSpacesFromEntities adjusts formatting entities to not start/end with spaces.
// If entity contains only spaces, it's removed from the list.
func trimSpacesFromEntities(source []rune, entities []tg.MessageEntityClass) []tg.MessageEntityClass {
	result := make([]tg.MessageEntityClass, 0, len(entities))

	for _, e := range entities {
		// Only process formatting entities
		switch entity := e.(type) {
		case *tg.MessageEntityBold:
			adjusted := trimEntitySpaces(source, entity.Offset, entity.Length)
			if adjusted != nil {
				result = append(result, &tg.MessageEntityBold{Offset: adjusted.offset, Length: adjusted.length})
			}
		case *tg.MessageEntityItalic:
			adjusted := trimEntitySpaces(source, entity.Offset, entity.Length)
			if adjusted != nil {
				result = append(result, &tg.MessageEntityItalic{Offset: adjusted.offset, Length: adjusted.length})
			}
		case *tg.MessageEntityStrike:
			adjusted := trimEntitySpaces(source, entity.Offset, entity.Length)
			if adjusted != nil {
				result = append(result, &tg.MessageEntityStrike{Offset: adjusted.offset, Length: adjusted.length})
			}
		case *tg.MessageEntityUnderline:
			adjusted := trimEntitySpaces(source, entity.Offset, entity.Length)
			if adjusted != nil {
				result = append(result, &tg.MessageEntityUnderline{Offset: adjusted.offset, Length: adjusted.length})
			}
		case *tg.MessageEntitySpoiler:
			adjusted := trimEntitySpaces(source, entity.Offset, entity.Length)
			if adjusted != nil {
				result = append(result, &tg.MessageEntitySpoiler{Offset: adjusted.offset, Length: adjusted.length})
			}
		default:
			// Keep other entities unchanged
			result = append(result, e)
		}
	}

	return result
}

type adjustedEntity struct {
	offset int
	length int
}

// trimEntitySpaces trims leading spaces from entity.
// Returns nil if entity becomes empty (only whitespace).
func trimEntitySpaces(source []rune, utf16Offset, utf16Length int) *adjustedEntity {
	// Convert to rune positions
	start := utf16OffsetToRune(string(source), utf16Offset)
	length := utf16LengthToRune(string(source), utf16Offset, utf16Length)
	end := start + length

	if start >= len(source) || end > len(source) {
		return nil
	}

	// Trim leading spaces (but not newlines)
	newStart := start
	for newStart < end && source[newStart] == ' ' {
		newStart++
	}

	// If all spaces, skip this entity
	allSpaces := true
	for i := newStart; i < end; i++ {
		if source[i] != ' ' && source[i] != '\t' {
			allSpaces = false
			break
		}
	}
	if allSpaces {
		return nil
	}

	// Calculate new UTF-16 offset and length
	newUtf16Offset := utf16Offset
	for i := start; i < newStart; i++ {
		newUtf16Offset += utf16RuneLen(source[i])
	}
	newUtf16Length := utf16Length
	for i := start; i < newStart; i++ {
		newUtf16Length -= utf16RuneLen(source[i])
	}

	if newUtf16Length <= 0 {
		return nil
	}

	return &adjustedEntity{offset: newUtf16Offset, length: newUtf16Length}
}

func writeCloseFormats(b *strings.Builder, fmt Format) {
	if fmt.Spoiler {
		b.WriteString("||")
	}
	if fmt.Underline {
		b.WriteString("</u>")
	}
	if fmt.Strikethrough {
		b.WriteString("~~")
	}
	if fmt.Bold {
		b.WriteString("**")
	}
	if fmt.Italic {
		b.WriteString("*")
	}
}

// utf16OffsetToRune converts UTF-16 offset to rune index.
func utf16OffsetToRune(s string, utf16Offset int) int {
	runeIdx := 0
	utf16Idx := 0
	for _, r := range s {
		if utf16Idx >= utf16Offset {
			break
		}
		utf16Idx += utf16RuneLen(r)
		runeIdx++
	}
	return runeIdx
}

// utf16LengthToRune converts UTF-16 length to rune count.
func utf16LengthToRune(s string, utf16Offset, utf16Length int) int {
	runeCount := 0
	utf16Idx := 0
	for _, r := range s {
		if utf16Idx >= utf16Offset+utf16Length {
			break
		}
		if utf16Idx >= utf16Offset {
			runeCount++
		}
		utf16Idx += utf16RuneLen(r)
	}
	return runeCount
}

// utf16RuneLen returns the size of a rune in UTF-16 code units.
func utf16RuneLen(r rune) int {
	if r >= 0x10000 {
		return 2 // surrogate pair
	}
	return 1
}
