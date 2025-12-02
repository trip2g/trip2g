package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

const (
	channelID = 1593462649
	messageID = 1053
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
	fmt.Println(Convert(msg))
}

func Convert(msg *tg.Message) string {
	source := []rune(msg.Message)

	// Collect replacements
	type replacement struct {
		offset int
		length int
		text   string
	}
	var replacements []replacement

	for _, e := range msg.Entities {
		offset, length := e.GetOffset(), e.GetLength()
		runeOffset := utf16OffsetToRune(msg.Message, offset)
		runeLength := utf16LengthToRune(msg.Message, offset, length)
		text := string(source[runeOffset : runeOffset+runeLength])

		var replText string
		switch entity := e.(type) {
		// case *tg.MessageEntityBold:
		// 	replText = wrapFormat(text, "**", "**")
		case *tg.MessageEntityItalic:
			replText = wrapFormat(text, "*", "*")
		// case *tg.MessageEntityUnderline:
		// 	replText = wrapFormat(text, "<u>", "</u>")
		// case *tg.MessageEntityStrike:
		// 	replText = wrapFormat(text, "~~", "~~")
		// case *tg.MessageEntityCode:
		// 	replText = "`" + text + "`"
		// case *tg.MessageEntityPre:
		// 	lang := entity.Language
		// 	replText = "```" + lang + "\n" + text + "\n```"
		case *tg.MessageEntityTextURL:
			// Trim trailing newlines from link text, but preserve them after
			linkText := strings.TrimRight(text, "\n")
			trailingNewlines := text[len(linkText):]
			replText = "[" + linkText + "](" + entity.URL + ")" + trailingNewlines
		case *tg.MessageEntityCustomEmoji:
			replText = fmt.Sprintf("![](https://ce.trip2g.com/%d.webp)", entity.DocumentID)
		default:
			continue
		}

		replacements = append(replacements, replacement{
			offset: runeOffset,
			length: runeLength,
			text:   replText,
		})
	}

	// Sort by offset in reverse order
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].offset > replacements[j].offset
	})

	// Apply replacements from the end
	for _, r := range replacements {
		source = append(source[:r.offset], append([]rune(r.text), source[r.offset+r.length:]...)...)
	}

	return string(source)
}

// wrapFormat wraps text in formatting, handling multiline lists
func wrapFormat(text, prefix, suffix string) string {
	if !strings.Contains(text, "\n") {
		return prefix + text + suffix
	}

	// Multiline - wrap each line separately
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Keep list marker outside formatting
		if strings.HasPrefix(line, "- ") {
			result = append(result, "- "+prefix+trimmed[2:]+suffix)
		} else {
			result = append(result, prefix+line+suffix)
		}
	}
	return strings.Join(result, "\n")
}

// utf16OffsetToRune converts UTF-16 offset to rune index
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

// utf16LengthToRune converts UTF-16 length to rune count
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

// utf16RuneLen returns the size of a rune in UTF-16 code units
func utf16RuneLen(r rune) int {
	if r >= 0x10000 {
		return 2 // surrogate pair
	}
	return 1
}

// func processMessage(msg *tg.Message) {
// 	fmt.Print(ConvertToMarkdownV2(msg))
// }

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
