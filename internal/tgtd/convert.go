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
		if correctOptions[string(answer.Option)] {
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
//
//nolint:gocognit,gocyclo,cyclop,funlen // complex entity processing with nested type switches
func convertText(msg *tg.Message) string {
	source := []rune(msg.Message)
	if len(source) == 0 {
		return ""
	}

	// Preprocess entities: trim leading/trailing spaces from formatting entities
	entities := trimSpacesFromEntities(source, msg.Entities)

	// Collect hashtag positions (need space before them, and no bold/italic inside)
	hashtagStarts := make(map[int]bool)
	hashtagPositions := make(map[int]bool)
	for _, e := range entities {
		if _, ok := e.(*tg.MessageEntityHashtag); ok {
			start := utf16OffsetToRune(msg.Message, e.GetOffset())
			length := utf16LengthToRune(msg.Message, e.GetOffset(), e.GetLength())
			hashtagStarts[start] = true
			for i := start; i < start+length && i < len(source); i++ {
				hashtagPositions[i] = true
			}
		}
	}

	// Build per-character format map
	formats := make([]Format, len(source))
	for _, e := range entities {
		start := utf16OffsetToRune(msg.Message, e.GetOffset())
		end := start + utf16LengthToRune(msg.Message, e.GetOffset(), e.GetLength())

		for i := start; i < end && i < len(formats); i++ {
			switch e.(type) {
			case *tg.MessageEntityBold:
				// Skip bold for hashtags - Obsidian doesn't support styled tags
				if !hashtagPositions[i] {
					formats[i].Bold = true
				}
			case *tg.MessageEntityItalic:
				// Skip italic for hashtags - Obsidian doesn't support styled tags
				if !hashtagPositions[i] {
					formats[i].Italic = true
				}
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
			// Cut at first newline - content after stays outside link
			linkText := text
			var rest string
			if idx := strings.Index(text, "\n"); idx != -1 {
				linkText = text[:idx]
				rest = text[idx:]
			}
			// Move leading emoji outside the link
			prefix, linkText := extractLeadingEmoji(linkText)
			// Escape = to avoid Setext header interpretation
			linkText = strings.ReplaceAll(linkText, "=", `\=`)
			replText = prefix + "[" + linkText + "](" + entity.URL + ")" + rest
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
			replText = fmt.Sprintf("![%s|20x20](https://ce.trip2g.com/%d.webp)", text, entity.DocumentID)
		case *tg.MessageEntityCode:
			replText = "`" + text + "`"
		case *tg.MessageEntityPre:
			// Code block - use enough backticks to avoid conflicts
			fence := codeFence(text)
			if entity.Language != "" {
				replText = fence + entity.Language + "\n" + text + "\n" + fence
			} else {
				replText = fence + "\n" + text + "\n" + fence
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

		// Add blank line before = at line start to avoid Setext header
		if atLineStart && r == '=' {
			result.WriteRune('\n')
		}

		atLineStart = false

		// Transition formatting
		//nolint:nestif // complex format transition with multiple open/close operations
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

		// Add space before hashtag if needed
		if hashtagStarts[i] && r == '#' && i > 0 {
			prev := source[i-1]
			if prev != ' ' && prev != '\n' {
				result.WriteRune(' ')
			}
		}

		// Escape markdown special chars to avoid interpretation
		switch r {
		case '*':
			result.WriteString(`\*`)
		case '_':
			result.WriteString(`\_`)
		case '`':
			result.WriteString("\\`")
		default:
			result.WriteRune(r)
		}
	}

	// Close any remaining formats
	writeCloseFormats(&result, currentFmt)

	out := result.String()

	// Fix nested format closing with trailing space: "** *" -> "*** "
	out = strings.ReplaceAll(out, "** *\n", "*** \n")
	out = strings.ReplaceAll(out, "* **\n", "*** \n")

	return out
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

// shouldTrimLeading returns true if rune should be trimmed from start of formatting.
// CommonMark: opening ** followed by punctuation doesn't work if preceded by word char.
func shouldTrimLeading(r rune) bool {
	switch r {
	case ' ', ',', '.', '!', '?', ';', ':', '—', '–', '-':
		return true
	}
	return false
}

// shouldTrimTrailing returns true if rune should be trimmed from end of formatting.
// Trim spaces, newlines, and mid-sentence punctuation (comma, semicolon, colon, dashes).
// Keep sentence-ending punctuation (.!?) inside bold as it looks better.
func shouldTrimTrailing(r rune) bool {
	switch r {
	case ' ', '\n', ',', ';', ':', '—', '–', '-':
		return true
	}
	return false
}

// trimEntitySpaces trims leading and trailing spaces/punctuation from entity.
// Returns nil if entity becomes empty.
// This is needed because CommonMark requires ** to be adjacent to word characters.
func trimEntitySpaces(source []rune, utf16Offset, utf16Length int) *adjustedEntity {
	// Convert to rune positions
	start := utf16OffsetToRune(string(source), utf16Offset)
	length := utf16LengthToRune(string(source), utf16Offset, utf16Length)
	end := start + length

	if start >= len(source) || end > len(source) {
		return nil
	}

	// Trim leading spaces and punctuation (but not newlines)
	newStart := start
	for newStart < end && shouldTrimLeading(source[newStart]) {
		newStart++
	}

	// Trim trailing spaces only (trailing punctuation is fine in CommonMark)
	newEnd := end
	for newEnd > newStart && shouldTrimTrailing(source[newEnd-1]) {
		newEnd--
	}

	// If all trimmed, skip this entity
	if newStart >= newEnd {
		return nil
	}

	// Calculate new UTF-16 offset and length
	newUtf16Offset := utf16Offset
	for i := start; i < newStart; i++ {
		newUtf16Offset += utf16RuneLen(source[i])
	}

	newUtf16Length := 0
	for i := newStart; i < newEnd; i++ {
		newUtf16Length += utf16RuneLen(source[i])
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

// extractLeadingEmoji extracts leading emoji from link text.
// Returns (prefix, remaining) where prefix should go outside the link.
func extractLeadingEmoji(text string) (string, string) {
	runes := []rune(text)
	if len(runes) == 0 {
		return "", text
	}

	first := runes[0]
	if !isEmoji(first) {
		return "", text
	}

	// Extract emoji + optional following space
	if len(runes) > 1 && runes[1] == ' ' {
		return string(runes[:2]), string(runes[2:])
	}
	return string(runes[:1]), string(runes[1:])
}

// isEmoji returns true if rune is likely an emoji.
func isEmoji(r rune) bool {
	// Emoji ranges
	switch {
	case r >= 0x2600 && r <= 0x26FF: // Misc Symbols
		return true
	case r >= 0x2700 && r <= 0x27BF: // Dingbats
		return true
	case r >= 0x1F300 && r <= 0x1F9FF: // Misc Symbols, Emoticons, etc.
		return true
	case r >= 0x1FA00 && r <= 0x1FAFF: // Extended-A
		return true
	}
	return false
}

// codeFence returns the appropriate fence string for a code block.
// Uses more backticks if the content contains backticks.
func codeFence(content string) string {
	maxBackticks := 0
	current := 0
	for _, r := range content {
		if r == '`' {
			current++
			if current > maxBackticks {
				maxBackticks = current
			}
		} else {
			current = 0
		}
	}
	// Use at least 3 backticks, or one more than the max found
	fenceLen := 3
	if maxBackticks >= 3 {
		fenceLen = maxBackticks + 1
	}
	return strings.Repeat("`", fenceLen)
}
