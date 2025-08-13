package tgbots

import (
	"context"
	"encoding/json"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerIO struct {
	dbBotID int64

	bot   *tgbotapi.BotAPI
	token string
}

func (io *HandlerIO) BotID() int64 {
	return io.dbBotID
}

func (io *HandlerIO) TgID() int64 {
	return io.bot.Self.ID
}

func (io *HandlerIO) BotLink() string {
	return fmt.Sprintf("https://t.me/%s", io.bot.Self.UserName)
}

func (io *HandlerIO) BotStartLink(param string) string {
	return fmt.Sprintf("https://t.me/%s?start=%s", io.bot.Self.UserName, param)
}

func (io *HandlerIO) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	return io.bot.Send(msg)
}

func (io *HandlerIO) Request(msg tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return io.bot.Request(msg)
}

func (io *HandlerIO) GetChatMemberStatus(ctx context.Context, chatID, userID int64) (string, error) {
	getChatMemberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	}

	resp, err := io.Request(getChatMemberConfig)
	if err != nil {
		return "", fmt.Errorf("failed to request chat member info: %w", err)
	}

	if !resp.Ok {
		return "", fmt.Errorf("telegram API error: %s", resp.Description)
	}

	var chatMember tgbotapi.ChatMember
	err = json.Unmarshal(resp.Result, &chatMember)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal chat member response: %w", err)
	}

	return chatMember.Status, nil
}

func (io *HandlerIO) GetBotCanInvite(ctx context.Context, chatID int64) (bool, error) {
	getChatMemberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: io.bot.Self.ID, // Check bot's own permissions
		},
	}

	resp, err := io.Request(getChatMemberConfig)
	if err != nil {
		return false, fmt.Errorf("failed to request bot chat member info: %w", err)
	}

	if !resp.Ok {
		return false, fmt.Errorf("telegram API error: %s", resp.Description)
	}

	var chatMember tgbotapi.ChatMember
	err = json.Unmarshal(resp.Result, &chatMember)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal bot chat member response: %w", err)
	}

	// Bot must be administrator to invite users
	if chatMember.Status != "administrator" {
		return false, nil
	}

	// Check if bot has invite_users permission
	// Note: In Telegram Bot API, CanInviteUsers defaults to true for administrators
	return chatMember.CanInviteUsers, nil
}

func (io *HandlerIO) InviteUserToChat(ctx context.Context, chatID, userID int64) error {
	// For private groups/supergroups, we need to create an invite link and send it to the user

	// First, try to create a one-time invite link
	createLinkConfig := tgbotapi.CreateChatInviteLinkConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: chatID,
		},
		MemberLimit: 1, // One-time use link
	}

	resp, err := io.Request(createLinkConfig)
	if err != nil {
		return fmt.Errorf("failed to create invite link: %w", err)
	}

	if !resp.Ok {
		return fmt.Errorf("telegram API error when creating invite link: %s", resp.Description)
	}

	// Parse the invite link from response
	var inviteLink tgbotapi.ChatInviteLink
	err = json.Unmarshal(resp.Result, &inviteLink)
	if err != nil {
		return fmt.Errorf("failed to unmarshal invite link response: %w", err)
	}

	// Send the invite link to the user
	message := fmt.Sprintf("🔗 Вот ваша пригласительная ссылка для входа в группу:\n\n%s\n\n⏰ Ссылка одноразовая и действует в течение ограниченного времени.", inviteLink.InviteLink)

	msg := tgbotapi.NewMessage(userID, message)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🚀 Войти в группу", inviteLink.InviteLink),
		),
	)

	_, err = io.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send invite link to user: %w", err)
	}

	return nil
}
