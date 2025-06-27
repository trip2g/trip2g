package handletgupdate

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	statusMember        = "member"
	statusAdministrator = "administrator"
	statusCreator       = "creator"
	statusLeft          = "left"
	statusKicked        = "kicked"
)

func (req *request) handleGroupAccess(ctx context.Context, args string) error {
	groupIDStr := strings.TrimPrefix(args, "group_")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		return req.SendMessage("Invalid group ID format")
	}

	userID := req.update.Message.From.ID

	params := db.InsertTgChatMemberParams{
		UserID: sql.NullInt64{Int64: userID, Valid: true},
		ChatID: sql.NullInt64{Int64: groupID, Valid: true},
	}

	err = req.env.InsertTgChatMember(ctx, params)
	if err != nil {
		req.env.Logger().Error(fmt.Sprintf("Failed to insert chat member: %v", err))
		return req.SendMessage("Error adding you to group access. Please try again later.")
	}

	text := fmt.Sprintf("✅ Access granted! You now have access to premium content from group %d.", groupID)

	tokenData := model.TgAuthToken{
		ChatID: userID,
		BotID:  req.env.BotID(),
	}

	authURL, err := req.env.GenerateTgAuthURL(ctx, "/", tokenData)
	if err != nil {
		return fmt.Errorf("failed to generate auth URL: %w", err)
	}

	msg := tgbotapi.NewMessage(req.update.Message.Chat.ID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Получить доступ", authURL),
		),
	)

	_, err = req.env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (req *request) handleMyChatMember(ctx context.Context) error {
	log := req.env.Logger()
	chatMember := req.update.MyChatMember

	// Only track channels, groups, and supergroups
	chat := chatMember.Chat
	if chat.Type != "channel" && chat.Type != "group" && chat.Type != "supergroup" {
		return nil
	}

	newStatus := chatMember.NewChatMember.Status
	oldStatus := chatMember.OldChatMember.Status

	log.Info("bot chat member status changed",
		"chat_id", chat.ID,
		"chat_type", chat.Type,
		"chat_title", chat.Title,
		"old_status", oldStatus,
		"new_status", newStatus,
	)

	// Bot was added to the chat
	if (newStatus == statusMember || newStatus == statusAdministrator) &&
		(oldStatus == statusLeft || oldStatus == statusKicked) {
		err := req.env.UpsertTgBotChat(ctx, db.UpsertTgBotChatParams{
			ID:        chat.ID,
			ChatType:  chat.Type,
			ChatTitle: chat.Title,
		})
		if err != nil {
			log.Error("failed to upsert bot chat", "error", err, "chat_id", chat.ID)
			return fmt.Errorf("failed to upsert bot chat: %w", err)
		}

		log.Info("bot added to chat", "chat_id", chat.ID, "chat_title", chat.Title)
	}

	// Bot was removed from the chat
	if (newStatus == statusLeft || newStatus == statusKicked) &&
		(oldStatus == statusMember || oldStatus == statusAdministrator) {
		err := req.env.MarkTgBotChatRemoved(ctx, chat.ID)
		if err != nil {
			log.Error("failed to mark bot chat as removed", "error", err, "chat_id", chat.ID)
			return fmt.Errorf("failed to mark bot chat as removed: %w", err)
		}

		log.Info("bot removed from chat", "chat_id", chat.ID, "chat_title", chat.Title)
	}

	return nil
}

func (req *request) handleChatMember(ctx context.Context) error { //nolint:unparam // always returns nil for now
	log := req.env.Logger()
	chatMember := req.update.ChatMember

	// Only track channels, groups, and supergroups
	chat := chatMember.Chat
	if chat.Type != "channel" && chat.Type != "group" && chat.Type != "supergroup" {
		return nil
	}

	userID := chatMember.NewChatMember.User.ID
	chatID := chat.ID
	newStatus := chatMember.NewChatMember.Status
	oldStatus := chatMember.OldChatMember.Status

	log.Info("user chat member status changed",
		"user_id", userID,
		"chat_id", chatID,
		"chat_type", chat.Type,
		"chat_title", chat.Title,
		"old_status", oldStatus,
		"new_status", newStatus,
	)

	// User joined the chat
	if (newStatus == statusMember || newStatus == statusAdministrator || newStatus == statusCreator) &&
		(oldStatus == statusLeft || oldStatus == statusKicked || oldStatus == "") {
		err := req.env.InsertTgChatMember(ctx, db.InsertTgChatMemberParams{
			UserID: sql.NullInt64{Int64: userID, Valid: true},
			ChatID: sql.NullInt64{Int64: chatID, Valid: true},
		})
		if err != nil {
			log.Error("failed to insert chat member", "error", err, "user_id", userID, "chat_id", chatID)
		} else {
			log.Info("user joined chat", "user_id", userID, "chat_id", chatID, "chat_title", chat.Title)
		}
	}

	// User left the chat
	if (newStatus == statusLeft || newStatus == statusKicked) &&
		(oldStatus == statusMember || oldStatus == statusAdministrator || oldStatus == statusCreator) {
		err := req.env.RemoveTgChatMember(ctx, db.RemoveTgChatMemberParams{
			UserID: sql.NullInt64{Int64: userID, Valid: true},
			ChatID: sql.NullInt64{Int64: chatID, Valid: true},
		})
		if err != nil {
			log.Error("failed to remove chat member", "error", err, "user_id", userID, "chat_id", chatID)
		} else {
			log.Info("user left chat", "user_id", userID, "chat_id", chatID, "chat_title", chat.Title)
		}
	}

	return nil
}

func (req *request) sendContentMenu(ctx context.Context) error {
	// generate list of active subgraphs

	return nil
}
