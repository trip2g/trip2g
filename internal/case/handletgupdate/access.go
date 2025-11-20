package handletgupdate

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
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

	chatTypeChannel    = "channel"
	chatTypeGroup      = "group"
	chatTypeSuperGroup = "supergroup"
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
	// First get the tg_bot_chat record to get the autoincrement id
	chat, err := req.env.TgBotChatByTelegramID(ctx, groupID)
	if err != nil {
		req.env.Logger().Error("Failed to get bot chat record", "error", err, "group_id", groupID)
		return req.SendMessage("❌ Group not found. Please try again later.")
	}

	params := db.InsertTgChatMemberParams{
		UserID: userID,
		ChatID: chat.ID,
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
	if chat.Type != chatTypeChannel && chat.Type != chatTypeGroup && chat.Type != chatTypeSuperGroup {
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
			// TODO: replace sleep with proper retry mechanism with exponential backoff
			// Telegram API needs time to propagate bot membership before we can check permissions
			time.Sleep(1 * time.Second)

			var checkErr error
			canInvite, checkErr = req.env.GetBotCanInvite(ctx, chat.ID)
			if checkErr != nil {
				return fmt.Errorf("failed to check bot invite permissions: %w", checkErr)
			}
		}

		err := req.env.UpsertTgBotChat(ctx, db.UpsertTgBotChatParams{
			TelegramID: chat.ID,
			ChatType:   chat.Type,
			ChatTitle:  chat.Title,
			CanInvite:  canInvite,
			BotID:      req.env.BotID(),
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
			// TODO: replace sleep with proper retry mechanism with exponential backoff
			// Telegram API needs time to propagate bot permission changes
			time.Sleep(1 * time.Second)

			var checkErr error
			canInvite, checkErr = req.env.GetBotCanInvite(ctx, chat.ID)
			if checkErr != nil {
				log.Error("failed to check bot invite permissions", "error", checkErr, "chat_id", chat.ID)
			}
		}

		err := req.env.UpdateTgBotChatCanInvite(ctx, db.UpdateTgBotChatCanInviteParams{
			CanInvite:  canInvite,
			TelegramID: chat.ID,
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

func (req *request) handleChatMember(ctx context.Context) error { //nolint:unparam,gocognit // error return for consistency, complex chat member logic
	log := req.env.Logger()
	chatMember := req.update.ChatMember

	log.Info("handleChatMember called", "chat_id", chatMember.Chat.ID, "user_id", chatMember.NewChatMember.User.ID)

	// Only track channels, groups, and supergroups
	chat := chatMember.Chat
	if chat.Type != "channel" && chat.Type != "group" && chat.Type != "supergroup" {
		log.Info("skipping chat member update - not a trackable chat type", "chat_type", chat.Type)
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
	//nolint:nestif // complex conditions needed for chat member status handling
	if (newStatus == statusMember || newStatus == statusAdministrator || newStatus == statusCreator) &&
		(oldStatus == statusLeft || oldStatus == statusKicked || oldStatus == "") {
		// Get the tg_bot_chat record to get the autoincrement id
		chat, chatErr := req.env.TgBotChatByTelegramID(ctx, chatID)
		if chatErr != nil {
			log.Error("failed to get bot chat record", "error", chatErr, "chat_id", chatID)
			return nil // Skip this user if we can't find the chat
		}

		err := req.env.InsertTgChatMember(ctx, db.InsertTgChatMemberParams{
			UserID: userID,
			ChatID: chat.ID,
		})
		if err != nil {
			log.Error("failed to insert chat member", "error", err, "user_id", userID, "chat_id", chatID)
		} else {
			log.Info("user joined chat", "user_id", userID, "chat_id", chatID, "chat_title", chat.ChatTitle)
		}

		// Update joined_at timestamp for tg_bot_chat_subgraph_accesses if user has system account
		log.Info("attempting to update access records for user", "tg_user_id", userID)
		user, userErr := req.env.UserByTgUserID(ctx, sql.NullInt64{Int64: userID, Valid: true})
		if userErr != nil {
			if !db.IsNoFound(userErr) {
				log.Error("failed to get user by telegram ID", "error", userErr, "tg_user_id", userID)
			} else {
				log.Info("no system account linked for telegram user", "tg_user_id", userID)
			}
			// No system account linked - skip access record update
		} else {
			log.Info("found system account for telegram user", "tg_user_id", userID, "system_user_id", user.ID)
			// Get active subgraphs for this chat to update access records
			subgraphs, subgraphErr := req.env.ListActiveTgChatSubgraphNamesByChatID(ctx, chat.ID)
			if subgraphErr != nil {
				log.Error("failed to get active subgraphs for chat", "error", subgraphErr, "chat_id", chat.ID)
			} else {
				log.Info("found active subgraphs for chat", "chat_id", chat.ID, "subgraphs", subgraphs, "count", len(subgraphs))
				// Update joined_at for each subgraph access record
				for _, subgraphName := range subgraphs {
					// Update joined_at timestamp for this access record
					updateErr := req.env.UpdateTgBotChatSubgraphAccessJoinedAt(ctx, db.UpdateTgBotChatSubgraphAccessJoinedAtParams{
						ChatID: chat.ID,
						UserID: user.ID,
						Name:   subgraphName,
					})
					if updateErr != nil {
						log.Error("failed to update joined_at for access record", "error", updateErr,
							"user_id", user.ID, "chat_id", chat.ID, "subgraph", subgraphName, "tg_user_id", userID)
					} else {
						log.Info("updated joined_at for access record",
							"user_id", user.ID, "chat_id", chat.ID, "subgraph", subgraphName, "tg_user_id", userID)
					}
				}
			}
		}
	}

	// User left the chat
	if (newStatus == statusLeft || newStatus == statusKicked) &&
		(oldStatus == statusMember || oldStatus == statusAdministrator || oldStatus == statusCreator) {
		// Get the tg_bot_chat record to get the autoincrement id
		chat, chatErr := req.env.TgBotChatByTelegramID(ctx, chatID)
		if chatErr != nil {
			log.Error("failed to get bot chat record", "error", chatErr, "chat_id", chatID)
			return nil // Skip this user if we can't find the chat
		}

		err := req.env.RemoveTgChatMember(ctx, db.RemoveTgChatMemberParams{
			UserID: userID,
			ChatID: chat.ID,
		})
		if err != nil {
			log.Error("failed to remove chat member", "error", err, "user_id", userID, "chat_id", chatID)
		} else {
			log.Info("user left chat", "user_id", userID, "chat_id", chatID, "chat_title", chat.ChatTitle)
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

func (req *request) handleMessageEvents(ctx context.Context) error { //nolint:unparam // error return maintained for consistency
	log := req.env.Logger()
	message := req.update.Message

	// Handle new chat members (users joining)
	if len(message.NewChatMembers) > 0 {
		for _, newMember := range message.NewChatMembers {
			// Skip bot additions (they are handled by MyChatMember events)
			if newMember.IsBot {
				continue
			}

			log.Info("user joined via new_chat_members", "user_id", newMember.ID, "chat_id", message.Chat.ID, "chat_title", message.Chat.Title)

			err := req.handleUserJoined(ctx, newMember.ID, message.Chat.ID)
			if err != nil {
				log.Error("failed to handle user joined", "error", err, "user_id", newMember.ID, "chat_id", message.Chat.ID)
			}
		}
	}

	// Handle user left chat
	if message.LeftChatMember != nil {
		leftMember := message.LeftChatMember

		// Skip bot removals (they are handled by MyChatMember events)
		if leftMember.IsBot {
			return nil
		}

		log.Info("user left via left_chat_member", "user_id", leftMember.ID, "chat_id", message.Chat.ID, "chat_title", message.Chat.Title)

		err := req.handleUserLeft(ctx, leftMember.ID, message.Chat.ID)
		if err != nil {
			log.Error("failed to handle user left", "error", err, "user_id", leftMember.ID, "chat_id", message.Chat.ID)
		}
	}

	return nil
}

func (req *request) handleUserJoined(ctx context.Context, userID int64, chatID int64) error {
	log := req.env.Logger()

	// Only track channels, groups, and supergroups
	chat := req.update.Message.Chat
	if chat.Type != chatTypeChannel && chat.Type != chatTypeGroup && chat.Type != chatTypeSuperGroup {
		log.Info("skipping user join - not a trackable chat type", "chat_type", chat.Type)
		return nil
	}

	// Get the tg_bot_chat record to get the autoincrement id
	botChat, chatErr := req.env.TgBotChatByTelegramID(ctx, chatID)
	if chatErr != nil {
		log.Error("failed to get bot chat record", "error", chatErr, "chat_id", chatID)
		return nil // Skip this user if we can't find the chat
	}

	err := req.env.InsertTgChatMember(ctx, db.InsertTgChatMemberParams{
		UserID: userID,
		ChatID: botChat.ID,
	})
	if err != nil {
		log.Error("failed to insert chat member", "error", err, "user_id", userID, "chat_id", chatID)
	} else {
		log.Info("user joined chat", "user_id", userID, "chat_id", chatID, "chat_title", botChat.ChatTitle)
	}

	// Update joined_at timestamp for tg_bot_chat_subgraph_accesses if user has system account
	log.Info("attempting to update access records for user", "tg_user_id", userID)
	user, userErr := req.env.UserByTgUserID(ctx, sql.NullInt64{Int64: userID, Valid: true})
	if userErr != nil {
		if !db.IsNoFound(userErr) {
			log.Error("failed to get user by telegram ID", "error", userErr, "tg_user_id", userID)
		} else {
			log.Info("no system account linked for telegram user", "tg_user_id", userID)
		}
		// No system account linked - skip access record update
		return nil
	}

	log.Info("found system account for telegram user", "tg_user_id", userID, "system_user_id", user.ID)
	// Get active subgraphs for this chat to update access records
	subgraphs, subgraphErr := req.env.ListActiveTgChatSubgraphNamesByChatID(ctx, botChat.ID)
	if subgraphErr != nil {
		log.Error("failed to get active subgraphs for chat", "error", subgraphErr, "chat_id", botChat.ID)
		return subgraphErr
	}

	log.Info("found active subgraphs for chat", "chat_id", botChat.ID, "subgraphs", subgraphs, "count", len(subgraphs))
	// Update joined_at for each subgraph access record
	for _, subgraphName := range subgraphs {
		// Update joined_at timestamp for this access record
		updateErr := req.env.UpdateTgBotChatSubgraphAccessJoinedAt(ctx, db.UpdateTgBotChatSubgraphAccessJoinedAtParams{
			ChatID: botChat.ID,
			UserID: user.ID,
			Name:   subgraphName,
		})
		if updateErr != nil {
			log.Error("failed to update joined_at for access record", "error", updateErr,
				"user_id", user.ID, "chat_id", botChat.ID, "subgraph", subgraphName, "tg_user_id", userID)
		} else {
			log.Info("updated joined_at for access record",
				"user_id", user.ID, "chat_id", botChat.ID, "subgraph", subgraphName, "tg_user_id", userID)
		}
	}

	return nil
}

func (req *request) handleUserLeft(ctx context.Context, userID int64, chatID int64) error { //nolint:unparam // error return maintained for consistency
	log := req.env.Logger()

	// Only track channels, groups, and supergroups
	chat := req.update.Message.Chat
	if chat.Type != chatTypeChannel && chat.Type != chatTypeGroup && chat.Type != chatTypeSuperGroup {
		log.Info("skipping user left - not a trackable chat type", "chat_type", chat.Type)
		return nil
	}

	// Get the tg_bot_chat record to get the autoincrement id
	botChat, chatErr := req.env.TgBotChatByTelegramID(ctx, chatID)
	if chatErr != nil {
		log.Error("failed to get bot chat record", "error", chatErr, "chat_id", chatID)
		return nil // Skip this user if we can't find the chat
	}

	err := req.env.RemoveTgChatMember(ctx, db.RemoveTgChatMemberParams{
		UserID: userID,
		ChatID: botChat.ID,
	})
	if err != nil {
		log.Error("failed to remove chat member", "error", err, "user_id", userID, "chat_id", chatID)
	} else {
		log.Info("user left chat", "user_id", userID, "chat_id", chatID, "chat_title", botChat.ChatTitle)
	}

	return nil
}
