package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
)

// FindTestChannels finds existing test channels by title.
func FindTestChannels(ctx context.Context, api *tg.Client) (map[string]ChannelConfig, error) {
	channels := make(map[string]ChannelConfig)

	channelTitles := map[string]string{
		ChannelBotScheduled:     "Trip2G Test Bot",
		ChannelBotInstant:       "Trip2G Test Bot Instant",
		ChannelAccountScheduled: "Trip2G Test Account",
		ChannelAccountInstant:   "Trip2G Test Account Instant",
	}

	var missing []string

	for name, title := range channelTitles {
		fmt.Printf("Looking for channel: %s... ", title)

		existing, err := findChannelByTitle(ctx, api, title)
		if err != nil {
			return nil, fmt.Errorf("failed to search for channel %s: %w", name, err)
		}

		if existing == nil {
			fmt.Println("NOT FOUND")
			missing = append(missing, title)
			continue
		}

		fmt.Printf("OK (ID=%d)\n", existing.ID)
		channels[name] = ChannelConfig{
			Title:      existing.Title,
			Username:   existing.Username,
			ID:         existing.ID,
			AccessHash: existing.AccessHash,
		}
	}

	if len(missing) > 0 {
		fmt.Println()
		fmt.Println("Please create the following channels manually:")
		for _, title := range missing {
			fmt.Printf("  - %s\n", title)
		}
		return nil, fmt.Errorf("%d channel(s) not found", len(missing))
	}

	return channels, nil
}

// ClearChannelMessages deletes all messages from a channel.
func ClearChannelMessages(ctx context.Context, api *tg.Client, channelID, accessHash int64) error {
	inputChannel := &tg.InputChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}

	peer := &tg.InputPeerChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}

	// Get all messages
	var messageIDs []int
	offsetID := 0

	for {
		history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     peer,
			Limit:    100,
			OffsetID: offsetID,
		})
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		var messages []tg.MessageClass
		switch h := history.(type) {
		case *tg.MessagesChannelMessages:
			messages = h.Messages
		case *tg.MessagesMessages:
			messages = h.Messages
		case *tg.MessagesMessagesSlice:
			messages = h.Messages
		}

		if len(messages) == 0 {
			break
		}

		for _, msg := range messages {
			switch m := msg.(type) {
			case *tg.Message:
				messageIDs = append(messageIDs, m.ID)
				offsetID = m.ID
			case *tg.MessageService:
				messageIDs = append(messageIDs, m.ID)
				offsetID = m.ID
			}
		}

		// If we got less than requested, we've reached the end
		if len(messages) < 100 {
			break
		}
	}

	if len(messageIDs) == 0 {
		return nil
	}

	// Delete in batches of 100
	for i := 0; i < len(messageIDs); i += 100 {
		end := i + 100
		if end > len(messageIDs) {
			end = len(messageIDs)
		}

		batch := messageIDs[i:end]
		_, err := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: inputChannel,
			ID:      batch,
		})
		if err != nil {
			return fmt.Errorf("failed to delete messages: %w", err)
		}

		// Small delay between batches
		if end < len(messageIDs) {
			time.Sleep(200 * time.Millisecond)
		}
	}

	return nil
}

// ClearAllTestChannels clears messages from all test channels.
func ClearAllTestChannels(ctx context.Context, api *tg.Client, channels map[string]ChannelConfig) error {
	for name, ch := range channels {
		fmt.Printf("Clearing channel %s (ID=%d)...\n", name, ch.ID)

		err := ClearChannelMessages(ctx, api, ch.ID, ch.AccessHash)
		if err != nil {
			return fmt.Errorf("failed to clear channel %s: %w", name, err)
		}

		fmt.Printf("  Cleared\n")
	}

	return nil
}

func findChannelByTitle(ctx context.Context, api *tg.Client, title string) (*tg.Channel, error) {
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return nil, err
	}

	var chats []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	}

	for _, chat := range chats {
		if ch, ok := chat.(*tg.Channel); ok {
			if ch.Title == title {
				return ch, nil
			}
		}
	}

	return nil, nil
}

// VerifyBotInChannels checks that bot is admin in bot channels.
func VerifyBotInChannels(ctx context.Context, api *tg.Client, channels map[string]ChannelConfig, botUsername string) error {
	// Only check bot channels
	botChannelNames := []string{ChannelBotScheduled, ChannelBotInstant}

	// Resolve bot user
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: botUsername,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve bot @%s: %w", botUsername, err)
	}

	if len(resolved.Users) == 0 {
		return fmt.Errorf("bot @%s not found", botUsername)
	}

	botUser, ok := resolved.Users[0].(*tg.User)
	if !ok {
		return fmt.Errorf("@%s is not a user", botUsername)
	}

	if !botUser.Bot {
		return fmt.Errorf("@%s is not a bot", botUsername)
	}

	fmt.Printf("Bot @%s found (ID: %d)\n", botUsername, botUser.ID)

	var missing []string

	for _, name := range botChannelNames {
		ch, ok := channels[name]
		if !ok {
			continue
		}

		fmt.Printf("Checking %s... ", ch.Title)

		// Get channel participants to check if bot is admin
		participants, err := api.ChannelsGetParticipant(ctx, &tg.ChannelsGetParticipantRequest{
			Channel: &tg.InputChannel{
				ChannelID:  ch.ID,
				AccessHash: ch.AccessHash,
			},
			Participant: &tg.InputPeerUser{
				UserID:     botUser.ID,
				AccessHash: botUser.AccessHash,
			},
		})
		if err != nil {
			fmt.Println("NOT FOUND")
			missing = append(missing, ch.Title)
			continue
		}

		// Check if bot is admin
		switch participants.Participant.(type) {
		case *tg.ChannelParticipantAdmin, *tg.ChannelParticipantCreator:
			fmt.Println("OK (admin)")
		default:
			fmt.Println("NOT ADMIN")
			missing = append(missing, ch.Title+" (not admin)")
		}
	}

	if len(missing) > 0 {
		fmt.Println()
		fmt.Println("Please add bot as admin to:")
		for _, title := range missing {
			fmt.Printf("  - %s\n", title)
		}
		return fmt.Errorf("bot not configured in %d channel(s)", len(missing))
	}

	return nil
}

// GetChannelMessages fetches messages from a channel.
func GetChannelMessages(ctx context.Context, api *tg.Client, channelID, accessHash int64, limit int) ([]*tg.Message, error) {
	peer := &tg.InputPeerChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}

	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	var result []*tg.Message

	var messages []tg.MessageClass
	switch h := history.(type) {
	case *tg.MessagesChannelMessages:
		messages = h.Messages
	case *tg.MessagesMessages:
		messages = h.Messages
	case *tg.MessagesMessagesSlice:
		messages = h.Messages
	}

	for _, msg := range messages {
		if m, ok := msg.(*tg.Message); ok {
			result = append(result, m)
		}
	}

	return result, nil
}
