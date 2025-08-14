package tgbots

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"trip2g/internal/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerIO struct {
	dbBotID int64
	logger  logger.Logger

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

	// First, unban the user in case they were previously banned (from expired subscription)
	err := io.UnbanChatMember(ctx, chatID, userID)
	if err != nil {
		// Log the unban error but don't fail - they might not be banned
		io.logger.Info("Failed to unban user before sending invite (user might not be banned)",
			"chatID", chatID, "userID", userID, "error", err.Error())
	}

	// Now, try to create a one-time invite link
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
	message := fmt.Sprintf("🔗 Вот ваша пригласительная ссылка для входа в группу:\n\n%s\n\n⏰ Ссылка одноразовая и действует в течение ограниченного времени.",
		inviteLink.InviteLink)

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

func (io *HandlerIO) KickChatMember(ctx context.Context, chatID, userID int64, chatType string) error {
	// Permanently ban user from chat when subscription ends
	// User will be unbanned when they request new access via /chats command

	banConfig := tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		RevokeMessages: false, // Don't delete user's messages
		// No UntilDate = permanent ban until manually unbanned
	}

	resp, err := io.Request(banConfig)
	if err != nil {
		// Check if this is the "group chat was upgraded" error at HTTP level
		if strings.Contains(err.Error(), "group chat was upgraded to a supergroup chat") {
			io.logger.Info("Group chat was upgraded to supergroup, cannot ban user from old chat ID",
				"chatID", chatID, "userID", userID)
			return nil
		}
		return fmt.Errorf("failed to ban chat member: %w", err)
	}

	if !resp.Ok {
		// Handle common recoverable Telegram API errors
		switch resp.Description {
		case "Bad Request: group chat was upgraded to a supergroup chat":
			// Group was upgraded - the old chat ID is no longer valid
			// Log this situation and return nil to continue with database cleanup
			io.logger.Info("Group chat was upgraded to supergroup, cannot ban user from old chat ID",
				"chatID", chatID, "userID", userID)
			return nil
		case "Bad Request: user not found":
			// User is no longer in the chat or doesn't exist
			return nil
		case "Bad Request: chat not found":
			// Chat no longer exists
			return nil
		case "Forbidden: bot was kicked from the group chat":
			// Bot was removed from chat - can't perform operations
			return nil
		case "Forbidden: bot is not a member of the supergroup chat":
			// Bot is not in the supergroup
			return nil
		default:
			return fmt.Errorf("telegram API error when banning user: %s", resp.Description)
		}
	}

	return nil
}

func (io *HandlerIO) UnbanChatMember(ctx context.Context, chatID, userID int64) error {
	// Unban user from chat (used when user requests new access)

	unbanConfig := tgbotapi.UnbanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		OnlyIfBanned: true, // Only unban if actually banned
	}

	resp, err := io.Request(unbanConfig)
	if err != nil {
		// Check if this is the "group chat was upgraded" error at HTTP level
		if strings.Contains(err.Error(), "group chat was upgraded to a supergroup chat") {
			io.logger.Info("Group chat was upgraded to supergroup, cannot unban user from old chat ID",
				"chatID", chatID, "userID", userID)
			return nil
		}
		return fmt.Errorf("failed to unban chat member: %w", err)
	}

	if !resp.Ok {
		// Handle common recoverable Telegram API errors
		switch resp.Description {
		case "Bad Request: group chat was upgraded to a supergroup chat":
			// Group was upgraded - the old chat ID is no longer valid
			return nil
		case "Bad Request: user not found":
			// User is no longer in the chat or doesn't exist
			return nil
		case "Bad Request: chat not found":
			// Chat no longer exists
			return nil
		case "Forbidden: bot was kicked from the group chat":
			// Bot was removed from chat - can't perform operations
			return nil
		case "Forbidden: bot is not a member of the supergroup chat":
			// Bot is not in the supergroup
			return nil
		default:
			return fmt.Errorf("telegram API error when unbanning user: %s", resp.Description)
		}
	}

	return nil
}
