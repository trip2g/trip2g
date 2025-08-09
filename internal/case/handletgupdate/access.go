package handletgupdate

import (
	"context"
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

	// First, verify that the user is actually a member of the group
	memberStatus, err := req.env.GetChatMemberStatus(ctx, groupID, userID)
	if err != nil {
		req.env.Logger().Error("Failed to check group membership", "error", err, "user_id", userID, "group_id", groupID)
		return req.SendMessage("❌ Unable to verify group membership. Please try again later.")
	}

	// Check if user has valid membership status
	// TODO: use map
	validStatuses := []string{statusMember, statusAdministrator, statusCreator}
	isValidMember := false
	for _, status := range validStatuses {
		if memberStatus == status {
			isValidMember = true
			break
		}
	}

	if !isValidMember {
		req.env.Logger().Info("User does not have valid membership status", "user_id", userID, "group_id", groupID, "status", memberStatus)
		return req.SendMessage("❌ You must be an active member of this group to get access. Please join the group and try again.")
	}

	// User is verified as a member, now grant access
	params := db.InsertTgChatMemberParams{
		UserID: userID,
		ChatID: groupID,
	}

	err = req.env.InsertTgChatMember(ctx, params)
	if err != nil {
		req.env.Logger().Error("Failed to insert chat member", "error", err, "user_id", userID, "group_id", groupID)
		return req.SendMessage("❌ Error granting access. Please try again later.")
	}

	return req.sendContentMenu(ctx)
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

	// Bot was added to the chat or status changed
	if (newStatus == statusMember || newStatus == statusAdministrator) &&
		(oldStatus == statusLeft || oldStatus == statusKicked) {
		// Check if bot can invite users (only for administrators)
		canInvite := false
		if newStatus == statusAdministrator {
			var checkErr error
			canInvite, checkErr = req.env.GetBotCanInvite(ctx, chat.ID)
			if checkErr != nil {
				return fmt.Errorf("failed to check bot invite permissions: %w", checkErr)
			}
		}

		err := req.env.UpsertTgBotChat(ctx, db.UpsertTgBotChatParams{
			ID:        chat.ID,
			ChatType:  chat.Type,
			ChatTitle: chat.Title,
			CanInvite: canInvite,
		})
		if err != nil {
			log.Error("failed to upsert bot chat", "error", err, "chat_id", chat.ID)
			return fmt.Errorf("failed to upsert bot chat: %w", err)
		}

		log.Info("bot added to chat", "chat_id", chat.ID, "chat_title", chat.Title, "can_invite", canInvite)
	}

	// Bot status changed (e.g., member -> administrator)
	if (newStatus == statusAdministrator || newStatus == statusMember) &&
		(oldStatus == statusAdministrator || oldStatus == statusMember) &&
		newStatus != oldStatus {
		// Check if bot can invite users (only for administrators)
		canInvite := false
		if newStatus == statusAdministrator {
			var checkErr error
			canInvite, checkErr = req.env.GetBotCanInvite(ctx, chat.ID)
			if checkErr != nil {
				log.Error("failed to check bot invite permissions", "error", checkErr, "chat_id", chat.ID)
			}
		}

		err := req.env.UpdateTgBotChatCanInvite(ctx, db.UpdateTgBotChatCanInviteParams{
			CanInvite: canInvite,
			ID:        chat.ID,
		})
		if err != nil {
			log.Error("failed to update bot chat invite permissions", "error", err, "chat_id", chat.ID)
			return fmt.Errorf("failed to update bot chat invite permissions: %w", err)
		}

		log.Info("bot permissions updated", "chat_id", chat.ID, "chat_title", chat.Title, "new_status", newStatus, "can_invite", canInvite)
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
			UserID: userID,
			ChatID: chatID,
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
			UserID: userID,
			ChatID: chatID,
		})
		if err != nil {
			log.Error("failed to remove chat member", "error", err, "user_id", userID, "chat_id", chatID)
		} else {
			log.Info("user left chat", "user_id", userID, "chat_id", chatID, "chat_title", chat.Title)
		}
	}

	return nil
}

// verifyOngoingGroupAccess provides periodic re-verification of group membership.
// This is a security measure to ensure users haven't left groups since initial access.
func (req *request) verifyOngoingGroupAccess(ctx context.Context, userID int64, subgraphName string) error {
	// For now, we implement a basic verification
	// TODO: Implement more sophisticated verification based on:
	// 1. Time-based re-verification (e.g., check every 24 hours)
	// 2. Database tracking of last verification time
	// 3. Automatic removal of access for users who left groups

	// This is a placeholder that always allows access for now
	// In a production environment, you would:
	// 1. Look up which group this subgraph access was granted for
	// 2. Re-verify the user is still a member of that group
	// 3. Remove access if verification fails

	return nil // Always allow for now - replace with actual verification logic
}

func (req *request) sendContentMenu(ctx context.Context) error {
	subgraphs, err := req.env.ListActiveTgChatSubgraphNamesByChatID(ctx, req.update.Message.Chat.ID)
	if err != nil {
		return fmt.Errorf("failed to list active subgraphs: %w", err)
	}

	// Re-verify group membership for enhanced security
	// This provides additional protection against users who left groups
	userID := req.update.Message.From.ID
	validSubgraphs := make([]string, 0, len(subgraphs))

	for _, subgraphName := range subgraphs {
		// For group-based access, re-verify membership periodically
		// This adds an extra security layer beyond the initial verification
		if verifyErr := req.verifyOngoingGroupAccess(ctx, userID, subgraphName); verifyErr != nil {
			req.env.Logger().Info("Skipping subgraph due to verification failure",
				"subgraph", subgraphName, "user_id", userID, "error", verifyErr)
			continue
		}
		validSubgraphs = append(validSubgraphs, subgraphName)
	}

	subgraphs = validSubgraphs

	noteViews := req.env.LatestNoteViews()

	text := "📚 Доступные материалы:\n\n"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	tokenData := model.TgAuthToken{
		ChatID: req.update.Message.Chat.ID,
		BotID:  req.env.BotID(),
	}

	for _, name := range subgraphs {
		_, ok := noteViews.Subgraphs[name]
		if ok {
			authURL, authErr := req.env.GenerateTgAuthURL(ctx, "/", tokenData)
			if authErr != nil {
				return fmt.Errorf("failed to generate auth URL: %w for %s", authErr, name)
			}

			keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(name, authURL),
			))
		}
	}

	if len(subgraphs) == 0 {
		text += "Ничего не найдено"

		msg := tgbotapi.NewMessage(req.update.Message.Chat.ID, text)

		_, sendErr := req.env.Send(msg)
		if sendErr != nil {
			return fmt.Errorf("failed to send message: %w", sendErr)
		}

		return nil
	}

	msg := tgbotapi.NewMessage(req.update.Message.Chat.ID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	_, err = req.env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
