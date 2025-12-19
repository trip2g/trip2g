package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/kr/pretty"
)

// MessageSnapshot represents a message for snapshot comparison.
type MessageSnapshot struct {
	ID        int    `json:"id"`
	Text      string `json:"text,omitempty"`
	Caption   string `json:"caption,omitempty"`
	HasMedia  bool   `json:"has_media,omitempty"`
	MediaType string `json:"media_type,omitempty"`
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
	err = os.MkdirAll(SnapshotDir, 0755)
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
			filename := filepath.Join(SnapshotDir, name+".json")
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
	fmt.Println("Snapshots saved to", SnapshotDir)
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
			filename := filepath.Join(SnapshotDir, name+".json")
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
			if !compareSnapshots(&expected, current) {
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

	snapshot := &ChannelSnapshot{
		ChannelName:  name,
		ChannelTitle: ch.Title,
		Messages:     make([]MessageSnapshot, 0, len(messages)),
	}

	// Process messages in reverse order (oldest first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		ms := MessageSnapshot{
			ID:   msg.ID,
			Text: msg.Message,
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
	default:
		return "unknown"
	}
}

func compareSnapshots(expected, actual *ChannelSnapshot) bool {
	if len(expected.Messages) != len(actual.Messages) {
		return false
	}

	for i := range expected.Messages {
		e := expected.Messages[i]
		a := actual.Messages[i]

		// Compare content, not IDs
		// Normalize text - trim leading/trailing whitespace to handle YAML block scalar differences
		if strings.TrimSpace(e.Text) != strings.TrimSpace(a.Text) {
			return false
		}
		if e.HasMedia != a.HasMedia {
			return false
		}
		if e.MediaType != a.MediaType {
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
