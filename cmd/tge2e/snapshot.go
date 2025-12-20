package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/kr/pretty"
)

// noteIDPattern matches "id: note_name" on first line of message text
var noteIDPattern = regexp.MustCompile(`^id: (\S+)`)

// extractNoteID extracts note ID from message text (first line: "id: note_name")
func extractNoteID(text string) string {
	matches := noteIDPattern.FindStringSubmatch(text)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

// MessageIDMap maps message_id -> note_id
type MessageIDMap map[int64]string

// buildMessageIDMap builds a map of message_id -> note_id from fetched messages
func buildMessageIDMap(messages []*tg.Message) MessageIDMap {
	result := make(MessageIDMap)
	for _, msg := range messages {
		noteID := extractNoteID(msg.Message)
		if noteID != "" {
			result[int64(msg.ID)] = noteID
		}
	}
	return result
}

// EntitySnapshot represents a message entity for snapshot comparison.
type EntitySnapshot struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url,omitempty"`
}

// MessageSnapshot represents a message for snapshot comparison.
type MessageSnapshot struct {
	Text      string           `json:"text,omitempty"`
	Entities  []EntitySnapshot `json:"entities,omitempty"`
	HasMedia  bool             `json:"has_media,omitempty"`
	MediaType string           `json:"media_type,omitempty"`
}

// ChannelSnapshot represents a channel's messages for snapshot comparison.
type ChannelSnapshot struct {
	ChannelName  string            `json:"channel_name"`
	ChannelTitle string            `json:"channel_title"`
	Messages     []MessageSnapshot `json:"messages"`
}

// runDump saves current channel messages to snapshot files.
func runDump() error {
	fmt.Println("=== Dumping Channel Messages ===")
	fmt.Println()

	creds, err := loadCredentials()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Ensure snapshot directory exists
	err = os.MkdirAll(snapshotDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	err = runWithClient(ctx, creds, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		for name, ch := range creds.Channels {
			fmt.Printf("Dumping %s...\n", ch.Title)

			snapshot, dumpErr := dumpChannel(ctx, api, name, ch)
			if dumpErr != nil {
				return fmt.Errorf("failed to dump %s: %w", name, dumpErr)
			}

			// Save to file as JSON
			filename := filepath.Join(snapshotDir, name+".json")
			data, marshalErr := json.MarshalIndent(snapshot, "", "  ")
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal snapshot: %w", marshalErr)
			}

			writeErr := os.WriteFile(filename, data, 0644)
			if writeErr != nil {
				return fmt.Errorf("failed to write snapshot: %w", writeErr)
			}

			fmt.Printf("  Saved %d messages to %s\n", len(snapshot.Messages), filename)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Snapshots saved to", snapshotDir)
	return nil
}

// runCheck compares current channel messages with saved snapshots.
func runCheck() error {
	fmt.Println("=== Checking Channel Messages ===")
	fmt.Println()

	creds, err := loadCredentials()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var failures []string

	err = runWithClient(ctx, creds, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		for name, ch := range creds.Channels {
			fmt.Printf("Checking %s... ", ch.Title)

			// Load expected snapshot
			filename := filepath.Join(snapshotDir, name+".json")
			expectedData, readErr := os.ReadFile(filename)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					fmt.Println("SKIP (no snapshot)")
					continue
				}
				return fmt.Errorf("failed to read snapshot: %w", readErr)
			}

			var expected ChannelSnapshot
			unmarshalErr := json.Unmarshal(expectedData, &expected)
			if unmarshalErr != nil {
				return fmt.Errorf("failed to parse snapshot: %w", unmarshalErr)
			}

			// Get current state
			current, dumpErr := dumpChannel(ctx, api, name, ch)
			if dumpErr != nil {
				return fmt.Errorf("failed to dump %s: %w", name, dumpErr)
			}

			// Compare messages (ignore timestamps and IDs)
			// For instant channels, normalize links (text_url/underline -> <u>)
			// because post order is non-deterministic: if post A links to post B,
			// the link resolves only if B was sent before A, otherwise it stays underlined
			isInstant := strings.Contains(name, "inst")
			if !compareSnapshots(&expected, current, isInstant) {
				fmt.Println("FAIL")
				failures = append(failures, name)
				printDiff(&expected, current)
			} else {
				fmt.Println("OK")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println()
	if len(failures) > 0 {
		fmt.Printf("Failed channels: %s\n", strings.Join(failures, ", "))
		return fmt.Errorf("%d channel(s) don't match snapshots", len(failures))
	}

	fmt.Println("All channels match snapshots.")
	return nil
}

func dumpChannel(ctx context.Context, api *tg.Client, name string, ch ChannelConfig) (*ChannelSnapshot, error) {
	messages, err := GetChannelMessages(ctx, api, ch.ID, ch.AccessHash, 100)
	if err != nil {
		return nil, err
	}

	// Build message_id -> note_id map from message texts (first line: "id: note_name")
	msgMap := buildMessageIDMap(messages)

	snapshot := &ChannelSnapshot{
		ChannelName:  name,
		ChannelTitle: ch.Title,
		Messages:     make([]MessageSnapshot, 0, len(messages)),
	}

	// Process messages in reverse order (oldest first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		ms := MessageSnapshot{
			Text: msg.Message,
		}

		// Extract entities
		if entities, ok := msg.GetEntities(); ok {
			for _, e := range entities {
				es := EntitySnapshot{
					Type: getEntityType(e),
				}
				// Get offset and length based on entity type
				switch ent := e.(type) {
				case *tg.MessageEntityTextURL:
					es.Offset = ent.Offset
					es.Length = ent.Length
					es.URL = ent.URL
				case *tg.MessageEntityURL:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityBold:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityItalic:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityUnderline:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityStrike:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityCode:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityPre:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityCustomEmoji:
					es.Offset = ent.Offset
					es.Length = ent.Length
				case *tg.MessageEntityBlockquote:
					es.Offset = ent.Offset
					es.Length = ent.Length
				}
				ms.Entities = append(ms.Entities, es)
			}
			// Normalize URLs in entities (replace message IDs with note IDs)
			ms.Entities = normalizeEntityURLs(ms.Entities, msgMap)
		}

		// Check for media
		if msg.Media != nil {
			ms.HasMedia = true
			ms.MediaType = getMediaType(msg.Media)
		}

		snapshot.Messages = append(snapshot.Messages, ms)
	}

	return snapshot, nil
}

func getEntityType(entity tg.MessageEntityClass) string {
	switch entity.(type) {
	case *tg.MessageEntityTextURL:
		return "text_url"
	case *tg.MessageEntityURL:
		return "url"
	case *tg.MessageEntityBold:
		return "bold"
	case *tg.MessageEntityItalic:
		return "italic"
	case *tg.MessageEntityUnderline:
		return "underline"
	case *tg.MessageEntityStrike:
		return "strikethrough"
	case *tg.MessageEntityCode:
		return "code"
	case *tg.MessageEntityPre:
		return "pre"
	case *tg.MessageEntityCustomEmoji:
		return "custom_emoji"
	case *tg.MessageEntityBlockquote:
		return "blockquote"
	default:
		return "unknown"
	}
}

func getMediaType(media tg.MessageMediaClass) string {
	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		return "photo"
	case *tg.MessageMediaDocument:
		if m.Document != nil {
			if doc, ok := m.Document.(*tg.Document); ok {
				for _, attr := range doc.Attributes {
					switch attr.(type) {
					case *tg.DocumentAttributeVideo:
						return "video"
					case *tg.DocumentAttributeAudio:
						return "audio"
					case *tg.DocumentAttributeSticker:
						return "sticker"
					case *tg.DocumentAttributeAnimated:
						return "animation"
					}
				}
			}
		}
		return "document"
	case *tg.MessageMediaGeo:
		return "geo"
	case *tg.MessageMediaContact:
		return "contact"
	case *tg.MessageMediaPoll:
		return "poll"
	case *tg.MessageMediaWebPage:
		return "webpage"
	default:
		return "unknown"
	}
}

// urlPattern matches Telegram channel message URLs like https://t.me/c/123456/789
var urlPattern = regexp.MustCompile(`https://t\.me/c/(\d+)/(\d+)`)

// normalizeEntityURLs replaces message IDs in URLs with note IDs.
// Changes https://t.me/c/123456/789 to https://t.me/c/{note_id}
// Extracts message ID from URL and looks up in msgMap built from message texts.
func normalizeEntityURLs(entities []EntitySnapshot, msgMap MessageIDMap) []EntitySnapshot {
	if msgMap == nil {
		return entities
	}

	result := make([]EntitySnapshot, len(entities))
	for i, e := range entities {
		result[i] = e
		if e.Type == "text_url" && e.URL != "" {
			matches := urlPattern.FindStringSubmatch(e.URL)
			if len(matches) == 3 {
				// matches[2] is the message ID
				var msgID int64
				fmt.Sscanf(matches[2], "%d", &msgID)

				if noteID, ok := msgMap[msgID]; ok {
					result[i].URL = fmt.Sprintf("https://t.me/c/{%s}", noteID)
				}
			}
		}
	}
	return result
}

// normalizeTextWithEntities replaces link text and underline text with placeholders.
// For instant channels, link resolution order is non-deterministic.
// Uses entities to precisely identify and replace link/underline spans.
func normalizeTextWithEntities(text string, entities []EntitySnapshot) string {
	if len(entities) == 0 {
		return text
	}

	// Convert to runes for proper Unicode handling
	runes := []rune(text)

	// Collect replacements (text_url -> <a>, underline -> <u>)
	// Sort by offset descending to replace from end to start
	type replacement struct {
		offset int
		length int
		tag    string
	}
	var replacements []replacement

	for _, e := range entities {
		switch e.Type {
		case "text_url", "underline":
			// Both resolved (text_url) and unresolved (underline) links become <u>
			replacements = append(replacements, replacement{e.Offset, e.Length, "<u>"})
		}
	}

	// Sort by offset descending
	for i := 0; i < len(replacements)-1; i++ {
		for j := i + 1; j < len(replacements); j++ {
			if replacements[j].offset > replacements[i].offset {
				replacements[i], replacements[j] = replacements[j], replacements[i]
			}
		}
	}

	// Apply replacements from end to start
	for _, r := range replacements {
		if r.offset >= 0 && r.offset+r.length <= len(runes) {
			newRunes := make([]rune, 0, len(runes))
			newRunes = append(newRunes, runes[:r.offset]...)
			newRunes = append(newRunes, []rune(r.tag)...)
			newRunes = append(newRunes, runes[r.offset+r.length:]...)
			runes = newRunes
		}
	}

	return string(runes)
}

// messageKey creates a comparable key for a message (order-independent comparison).
func messageKey(m MessageSnapshot) string {
	return fmt.Sprintf("%s|%v|%s", strings.TrimSpace(m.Text), m.HasMedia, m.MediaType)
}

// messageKeyNormalized creates a key with normalized links/underlines for instant channels.
func messageKeyNormalized(m MessageSnapshot) string {
	text := normalizeTextWithEntities(m.Text, m.Entities)
	return fmt.Sprintf("%s|%v|%s", strings.TrimSpace(text), m.HasMedia, m.MediaType)
}

func compareSnapshots(expected, actual *ChannelSnapshot, instant bool) bool {
	if len(expected.Messages) != len(actual.Messages) {
		return false
	}

	// Choose key function based on channel type
	keyFunc := messageKey
	if instant {
		keyFunc = messageKeyNormalized
	}

	// Count expected messages
	expectedCounts := make(map[string]int)
	for _, m := range expected.Messages {
		expectedCounts[keyFunc(m)]++
	}

	// Count actual messages
	actualCounts := make(map[string]int)
	for _, m := range actual.Messages {
		actualCounts[keyFunc(m)]++
	}

	// Compare counts
	for key, count := range expectedCounts {
		if actualCounts[key] != count {
			return false
		}
	}

	return true
}

func printDiff(expected, actual *ChannelSnapshot) {
	fmt.Println()
	fmt.Println("Expected:")
	for _, m := range expected.Messages {
		printMessage(m)
	}
	fmt.Println()
	fmt.Println("Actual:")
	for _, m := range actual.Messages {
		printMessage(m)
	}
	fmt.Println()

	// Use pretty for detailed diff
	diff := pretty.Diff(expected.Messages, actual.Messages)
	if len(diff) > 0 {
		fmt.Println("Diff:")
		for _, d := range diff {
			fmt.Printf("  %s\n", d)
		}
	}
}

func printMessage(m MessageSnapshot) {
	text := m.Text
	if len(text) > 50 {
		text = text[:50] + "..."
	}
	if m.HasMedia {
		fmt.Printf("  [%s] %s\n", m.MediaType, text)
	} else {
		fmt.Printf("  %s\n", text)
	}
}
