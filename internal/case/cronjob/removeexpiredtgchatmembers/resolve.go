package removeexpiredtgchatmembers

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type accessRow = db.ListTgBotChatSubgraphAccessesRow

type Env interface {
	Logger() logger.Logger
	AuditLogger() logger.Logger
	ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error)
	ListTgBotChatSubgraphAccesses(ctx context.Context, filter db.ListTgBotChatSubgraphAccessesParams) ([]accessRow, error)
	DeleteTgBotChatSubgraphAccess(ctx context.Context, arg db.DeleteTgBotChatSubgraphAccessParams) error
	RemoveTgChatMember(ctx context.Context, arg db.RemoveTgChatMemberParams) error
	KickTelegramChatMember(ctx context.Context, chatID, userID int64) error
	UserByID(ctx context.Context, id int64) (db.User, error)
	SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error)
}

type Filter struct {
	UserID *int64
	ChatID *int64
}

type Result struct {
	RemovedCount int

	Errors []error
}

func Resolve(ctx context.Context, env Env, filter Filter) (*Result, error) {
	accesses := map[int64][]*accessRow{}

	filterParams := db.ListTgBotChatSubgraphAccessesParams{
		UserID: filter.UserID,
		ChatID: filter.ChatID,
	}

	rows, err := env.ListTgBotChatSubgraphAccesses(ctx, filterParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list tg bot chat subgraph accesses: %w", err)
	}

	for _, row := range rows {
		userID := row.TgBotChatSubgraphAccess.UserID
		accesses[userID] = append(accesses[userID], &row)
	}

	log := logger.WithPrefix(env.Logger(), "removeexpiredtgchatmembers:")
	result := Result{}

	for userID, accessRows := range accesses {
		removedCount, processErr := processUser(ctx, env, userID, accessRows)
		if processErr != nil {
			log.Error("failed to process user", "userID", userID, "error", processErr)
			env.AuditLogger().Error("failed to remove expired tg chat member", "userID", userID, "error", processErr)
			result.Errors = append(result.Errors, processErr)
		}

		if removedCount == 0 {
			continue
		}

		result.RemovedCount += removedCount

		env.AuditLogger().Info("remove expired tg chat member", "userID", userID)
	}

	return &result, nil
}

func processUser(ctx context.Context, env Env, userID int64, accesses []*db.ListTgBotChatSubgraphAccessesRow) (int, error) {
	subgraphMap, err := getUserSubgraphs(ctx, env, userID)
	if err != nil {
		return 0, err
	}

	// Get user info for notifications
	user, err := env.UserByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user by ID %d: %w", userID, err)
	}

	count := 0

	for _, access := range accesses {
		_, haveAccess := subgraphMap[access.Subgraph.Name]
		if haveAccess {
			continue
		}

		env.Logger().Debug("processing expired access",
			"userID", userID,
			"chatID", access.TgBotChatSubgraphAccess.ChatID,
		)

		processErr := processExpiredAccess(ctx, env, &user, access)
		if processErr != nil {
			// Continue processing other accesses even if one fails
			// Return error that includes details about which access failed
			return 0, fmt.Errorf("failed to process expired access for user %d in chat %d (subgraph %s): %w",
				userID, access.TgBotChatSubgraphAccess.ChatID, access.Subgraph.Name, processErr)
		}

		count++
	}

	return count, nil
}

func processExpiredAccess(ctx context.Context, env Env, user *db.User, access *db.ListTgBotChatSubgraphAccessesRow) error {
	// The access.TgBotChatSubgraphAccess.ChatID is already the internal chat ID
	// We can use the chat info from the access row which includes the telegram ID
	chat := access.TgBotChat

	// First, permanently ban the user from the actual Telegram chat if they have a telegram ID
	// They will be unbanned when they request new access via /chats command
	if user.TgUserID != nil {
		err := env.KickTelegramChatMember(ctx, chat.ID, user.ID)
		if err != nil {
			return fmt.Errorf("failed to ban user from Telegram chat (userID: %d, chatID: %d): %w",
				user.ID, chat.ID, err)
		}
	}

	// Remove user from the chat database record
	var tgUserID int64
	if user.TgUserID != nil {
		tgUserID = *user.TgUserID
	}
	err := env.RemoveTgChatMember(ctx, db.RemoveTgChatMemberParams{
		UserID: tgUserID,
		ChatID: chat.TelegramID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove chat member from database (telegramUserID: %d, chatID: %d): %w",
			tgUserID, chat.TelegramID, err)
	}

	// Send notification to user if we have their telegram ID
	if user != nil && user.TgUserID != nil {
		notifyErr := sendExpirationNotification(ctx, env, chat.ID, *user.TgUserID, chat.ChatTitle, access.Subgraph.Name)
		if notifyErr != nil {
			// Don't fail the whole operation if notification fails
			// Just wrap the error for context
			return fmt.Errorf("failed to send expiration notification to telegram user %d: %w",
				*user.TgUserID, notifyErr)
		}
	}

	// Remove the access record from the database
	err = env.DeleteTgBotChatSubgraphAccess(ctx, db.DeleteTgBotChatSubgraphAccessParams{
		ChatID:     access.TgBotChatSubgraphAccess.ChatID,
		UserID:     access.TgBotChatSubgraphAccess.UserID,
		SubgraphID: access.TgBotChatSubgraphAccess.SubgraphID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete access record: %w", err)
	}

	return nil
}

func sendExpirationNotification(ctx context.Context, env Env, chatID int64, telegramUserID int64, chatTitle, subgraphName string) error {
	message := fmt.Sprintf(
		"⚠️ Ваш доступ к группе \"%s\" (подписка: %s) истёк.\n\n"+
			"Для продления доступа обновите подписку.",
		chatTitle, subgraphName)

	msg := tgbotapi.NewMessage(telegramUserID, message)
	_, err := env.SendTelegramMessage(ctx, chatID, msg)

	if err != nil {
		return fmt.Errorf("failed to send telegram message to user: %w", err)
	}

	return nil
}

func getUserSubgraphs(ctx context.Context, env Env, userID int64) (map[string]struct{}, error) {
	subgraphs, err := env.ListActiveUserSubgraphs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active user subgraphs: %w", err)
	}

	subgraphMap := make(map[string]struct{}, len(subgraphs))

	for _, subgraph := range subgraphs {
		subgraphMap[subgraph] = struct{}{}
	}

	return subgraphMap, nil
}
