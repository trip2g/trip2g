package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

type PostRecord struct {
	MessageID int    `json:"message_id"`
	Filename  string `json:"filename"`
}

type PostsTracking struct {
	ChannelID string       `json:"channel_id"`
	Posts     []PostRecord `json:"posts"`
}

func main() {
	var channelID int64
	var outputPath string
	var inputPath string
	var limit int
	var step int

	flag.Int64Var(&channelID, "channel-id", 0, "Telegram channel ID (numeric)")
	flag.StringVar(&outputPath, "output-path", "", "Output directory path for markdown files")
	flag.StringVar(&inputPath, "input-path", "", "Input directory path (for step1+)")
	flag.IntVar(&limit, "limit", 10, "Maximum number of messages to export")
	flag.IntVar(&step, "step", 0, "Pipeline step: 0=export, 1=titles, 2=wikilinks")
	flag.Parse()

	// Step 1: Transform titles (no Telegram needed)
	if step == 1 {
		if inputPath == "" || outputPath == "" {
			log.Fatal("--input-path and --output-path are required for step 1")
		}
		err := runStep1(inputPath, outputPath)
		if err != nil {
			log.Fatalf("Step 1 failed: %v", err)
		}
		return
	}

	// Step 2: Replace telegram links with wikilinks
	if step == 2 {
		if inputPath == "" || outputPath == "" {
			log.Fatal("--input-path and --output-path are required for step 2")
		}
		err := runStep2(inputPath, outputPath)
		if err != nil {
			log.Fatalf("Step 2 failed: %v", err)
		}
		return
	}

	// Step 0: Export from Telegram
	if channelID == 0 {
		log.Fatal("--channel-id is required")
	}
	if outputPath == "" {
		log.Fatal("--output-path is required")
	}

	// Read API credentials from environment
	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	apiHashStr := os.Getenv("TELEGRAM_API_HASH")
	if apiIDStr == "" || apiHashStr == "" {
		log.Fatal("TELEGRAM_API_ID and TELEGRAM_API_HASH environment variables are required")
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_API_ID: %v", err)
	}

	// Create output directory if it doesn't exist
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	ctx := context.Background()

	// Initialize Telegram client
	client := telegram.NewClient(apiID, apiHashStr, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: "session.json",
		},
	})

	err = client.Run(ctx, func(ctx context.Context) error {
		// Authenticate
		err := authenticate(ctx, client)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// List available channels
		err = listChannels(ctx, client.API())
		if err != nil {
			return fmt.Errorf("failed to list channels: %w", err)
		}

		// Export channel
		err = exportChannel(ctx, client, channelID, outputPath, limit)
		if err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Client error: %v", err)
	}

	log.Printf("Export completed successfully to %s", outputPath)
}

func listChannels(ctx context.Context, api *tg.Client) error {
	// Fetch all dialogs
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return fmt.Errorf("failed to get dialogs: %w", err)
	}

	var chats []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	}

	log.Println("\n=== Available Channels ===")
	channelCount := 0
	for _, chat := range chats {
		if channel, ok := chat.(*tg.Channel); ok {
			channelCount++
			title := channel.Title
			if channel.Username != "" {
				log.Printf("Channel ID: %d | @%s | %s", channel.ID, channel.Username, title)
			} else {
				log.Printf("Channel ID: %d | %s", channel.ID, title)
			}
		}
	}
	log.Printf("=== Total: %d channels ===\n", channelCount)

	return nil
}

type terminalAuth struct {
	phone string
}

func (a *terminalAuth) Phone(_ context.Context) (string, error) {
	return a.phone, nil
}

func (a *terminalAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	var password string
	_, err := fmt.Scanln(&password)
	return password, err
}

func (a *terminalAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code: ")
	var code string
	_, err := fmt.Scanln(&code)
	return code, err
}

func (a *terminalAuth) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func (a *terminalAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("signup not supported")
}

func authenticate(ctx context.Context, client *telegram.Client) error {
	status, err := client.Auth().Status(ctx)
	if err != nil {
		return err
	}

	if !status.Authorized {
		// Interactive authentication - prompt for phone
		fmt.Print("Enter phone number (international format, e.g., +1234567890): ")
		var phone string
		_, err := fmt.Scanln(&phone)
		if err != nil {
			return fmt.Errorf("failed to read phone: %w", err)
		}

		flow := auth.NewFlow(
			&terminalAuth{phone: phone},
			auth.SendCodeOptions{},
		)

		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}
	}

	return nil
}

func loadPostsTracking(outputPath string, channelID int64) (*PostsTracking, error) {
	trackingPath := filepath.Join(outputPath, "posts.json")

	data, err := os.ReadFile(trackingPath)
	if os.IsNotExist(err) {
		// File doesn't exist yet, create new tracking
		return &PostsTracking{
			ChannelID: fmt.Sprintf("%d", channelID),
			Posts:     []PostRecord{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking file: %w", err)
	}

	var tracking PostsTracking
	err = json.Unmarshal(data, &tracking)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tracking file: %w", err)
	}

	return &tracking, nil
}

func savePostsTracking(outputPath string, tracking *PostsTracking) error {
	trackingPath := filepath.Join(outputPath, "posts.json")

	data, err := json.MarshalIndent(tracking, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tracking: %w", err)
	}

	err = os.WriteFile(trackingPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write tracking file: %w", err)
	}

	return nil
}

func isMessageProcessed(tracking *PostsTracking, messageID int) bool {
	for _, post := range tracking.Posts {
		if post.MessageID == messageID {
			return true
		}
	}
	return false
}

func exportChannel(ctx context.Context, client *telegram.Client, channelID int64, outputPath string, limit int) error {
	api := client.API()

	// TODO: Tracking disabled for testing
	// tracking, err := loadPostsTracking(outputPath, channelID)
	// if err != nil {
	// 	return fmt.Errorf("failed to load tracking: %w", err)
	// }
	// log.Printf("Loaded tracking file: %d posts already processed", len(tracking.Posts))

	// Get channel
	channel, err := getChannel(ctx, api, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Fetch messages with pagination (Telegram limits to 100 per request)
	var msgList []*tg.Message
	offsetID := 0
	batchSize := 100

	for len(msgList) < limit {
		remaining := limit - len(msgList)
		fetchLimit := batchSize
		if remaining < batchSize {
			fetchLimit = remaining
		}

		messages, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     &tg.InputPeerChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash},
			Limit:    fetchLimit,
			OffsetID: offsetID,
		})
		if err != nil {
			return fmt.Errorf("failed to fetch messages: %w", err)
		}

		var batch []*tg.Message
		switch m := messages.(type) {
		case *tg.MessagesChannelMessages:
			for _, msg := range m.Messages {
				if message, ok := msg.(*tg.Message); ok {
					batch = append(batch, message)
				}
			}
		case *tg.MessagesMessages:
			for _, msg := range m.Messages {
				if message, ok := msg.(*tg.Message); ok {
					batch = append(batch, message)
				}
			}
		}

		if len(batch) == 0 {
			break // No more messages
		}

		msgList = append(msgList, batch...)
		offsetID = batch[len(batch)-1].ID
		log.Printf("Fetched %d messages (total: %d)", len(batch), len(msgList))
	}

	log.Printf("Total fetched: %d messages", len(msgList))

	// Process messages
	processedCount := 0
	failedCount := 0
	for i, msg := range msgList {
		log.Printf("=== Processing %d/%d ===", i+1, len(msgList))
		filename, err := processMessage(ctx, msg, channelID, outputPath)
		if err != nil {
			log.Printf("❌ Failed to process message %d: %v", msg.ID, err)
			failedCount++
			continue
		}

		if filename != "" {
			processedCount++
			log.Printf("✓ Successfully exported: %s", filename)
		}
	}

	// TODO: Tracking disabled for testing
	// err = savePostsTracking(outputPath, tracking)
	// if err != nil {
	// 	return fmt.Errorf("failed to save tracking: %w", err)
	// }

	log.Printf("✓ Successfully processed: %d, ❌ Failed: %d, Total: %d", processedCount, failedCount, len(msgList))

	return nil
}

func getChannel(ctx context.Context, api *tg.Client, channelID int64) (*tg.Channel, error) {
	// Fetch all dialogs to find the channel with proper access hash
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	var chats []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	}

	// Find the channel by ID
	for _, chat := range chats {
		if channel, ok := chat.(*tg.Channel); ok && channel.ID == channelID {
			return channel, nil
		}
	}

	return nil, fmt.Errorf("channel not found in your dialogs (make sure you're a member)")
}

func processMessage(ctx context.Context, msg *tg.Message, channelID int64, outputPath string) (string, error) {
	// Skip empty messages
	if msg.Message == "" {
		log.Printf("Skipping message %d: empty", msg.ID)
		return "", nil
	}

	log.Printf("Processing message %d (length: %d chars)", msg.ID, len(msg.Message))

	// Save raw JSON for debugging
	jsonPath := filepath.Join(outputPath, fmt.Sprintf("%d.json", msg.ID))
	jsonData, _ := json.MarshalIndent(msg, "", "  ")
	os.WriteFile(jsonPath, jsonData, 0644)

	// Convert message to markdown
	markdown, frontmatter := ConvertToMarkdown(msg, channelID)

	// Filename is just the message ID
	filename := fmt.Sprintf("%d.md", msg.ID)

	// Create full markdown with frontmatter
	fullContent := formatWithFrontmatter(frontmatter, markdown)

	// Write to file
	filePath := filepath.Join(outputPath, filename)
	err := os.WriteFile(filePath, []byte(fullContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Exported message %d: %s", msg.ID, filename)
	return filename, nil
}

func formatWithFrontmatter(frontmatter map[string]interface{}, content string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	for key, value := range frontmatter {
		sb.WriteString(fmt.Sprintf("%s: %v\n", key, formatYAMLValue(value)))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(content)

	return sb.String()
}

func formatYAMLValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Quote strings that contain special characters or look like numbers
		if strings.ContainsAny(v, ":{}[]!#|>&*") || v == "" {
			return fmt.Sprintf("%q", v)
		}
		// Also quote if it starts with a number or looks like a boolean
		if len(v) > 0 && (v[0] >= '0' && v[0] <= '9' || v == "true" || v == "false" || v == "yes" || v == "no") {
			return fmt.Sprintf("%q", v)
		}
		return v
	case int, int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
