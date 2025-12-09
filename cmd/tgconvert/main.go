package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"trip2g/internal/tgtd"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

const (
	channelID = 1593462649
	messageID = 2079
)

func main() {
	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	apiHashStr := os.Getenv("TELEGRAM_API_HASH")
	if apiIDStr == "" || apiHashStr == "" {
		log.Fatal("TELEGRAM_API_ID and TELEGRAM_API_HASH environment variables are required")
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_API_ID: %v", err)
	}

	ctx := context.Background()

	client := telegram.NewClient(apiID, apiHashStr, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: "session.json",
		},
	})

	err = client.Run(ctx, func(ctx context.Context) error {
		err := authenticate(ctx, client)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		api := client.API()

		// Get channel
		channel, err := getChannel(ctx, api, channelID)
		if err != nil {
			return fmt.Errorf("failed to get channel: %w", err)
		}

		// Fetch specific message
		messages, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash},
			ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
		})
		if err != nil {
			return fmt.Errorf("failed to fetch message: %w", err)
		}

		var msg *tg.Message
		switch m := messages.(type) {
		case *tg.MessagesChannelMessages:
			if len(m.Messages) > 0 {
				if message, ok := m.Messages[0].(*tg.Message); ok {
					msg = message
				}
			}
		}

		if msg == nil {
			return fmt.Errorf("message not found")
		}

		// Process the message
		processMessage(msg)

		return nil
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func processMessage(msg *tg.Message) {
	// Debug: print message text as Go string
	fmt.Printf("Message: %q\n\n", msg.Message)

	// Debug: print poll data if present
	if msg.Media != nil {
		if poll, ok := msg.Media.(*tg.MessageMediaPoll); ok {
			fmt.Println("Poll:")
			fmt.Printf("  Question: %q\n", poll.Poll.Question.Text)
			fmt.Printf("  Quiz: %v\n", poll.Poll.Quiz)
			fmt.Println("  Answers:")
			for i, a := range poll.Poll.Answers {
				fmt.Printf("    [%d] Option: %v, Text: %q\n", i, a.Option, a.Text.Text)
			}
			fmt.Println("  Results:")
			if poll.Results.Results != nil {
				for i, r := range poll.Results.Results {
					fmt.Printf("    [%d] Option: %v, Correct: %v, Chosen: %v, Voters: %d\n",
						i, r.Option, r.Correct, r.Chosen, r.Voters)
				}
			} else {
				fmt.Println("    (no results)")
			}
			fmt.Println()
		}
	}

	// Debug: print entities as Go code for tests
	fmt.Println("Entities: []tg.MessageEntityClass{")
	for _, e := range msg.Entities {
		switch entity := e.(type) {
		case *tg.MessageEntityBold:
			fmt.Printf("\t&tg.MessageEntityBold{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntityItalic:
			fmt.Printf("\t&tg.MessageEntityItalic{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntityStrike:
			fmt.Printf("\t&tg.MessageEntityStrike{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntityUnderline:
			fmt.Printf("\t&tg.MessageEntityUnderline{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntitySpoiler:
			fmt.Printf("\t&tg.MessageEntitySpoiler{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntityCode:
			fmt.Printf("\t&tg.MessageEntityCode{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntityPre:
			fmt.Printf("\t&tg.MessageEntityPre{Offset: %d, Length: %d, Language: %q},\n", entity.Offset, entity.Length, entity.Language)
		case *tg.MessageEntityTextURL:
			fmt.Printf("\t&tg.MessageEntityTextURL{Offset: %d, Length: %d, URL: %q},\n", entity.Offset, entity.Length, entity.URL)
		case *tg.MessageEntityURL:
			fmt.Printf("\t&tg.MessageEntityURL{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntityMention:
			fmt.Printf("\t&tg.MessageEntityMention{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		case *tg.MessageEntityMentionName:
			fmt.Printf("\t&tg.MessageEntityMentionName{Offset: %d, Length: %d, UserID: %d},\n", entity.Offset, entity.Length, entity.UserID)
		case *tg.MessageEntityCustomEmoji:
			fmt.Printf("\t&tg.MessageEntityCustomEmoji{Offset: %d, Length: %d, DocumentID: %d},\n", entity.Offset, entity.Length, entity.DocumentID)
		case *tg.MessageEntityHashtag:
			fmt.Printf("\t&tg.MessageEntityHashtag{Offset: %d, Length: %d},\n", entity.Offset, entity.Length)
		default:
			fmt.Printf("\t// unknown entity: %T\n", e)
		}
	}
	fmt.Println("}")

	fmt.Println("\n--- Result ---")
	fmt.Println(tgtd.Convert(msg))
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

func getChannel(ctx context.Context, api *tg.Client, channelID int64) (*tg.Channel, error) {
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

	for _, chat := range chats {
		if channel, ok := chat.(*tg.Channel); ok && channel.ID == channelID {
			return channel, nil
		}
	}

	return nil, fmt.Errorf("channel not found")
}
