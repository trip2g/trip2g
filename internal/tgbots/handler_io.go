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
