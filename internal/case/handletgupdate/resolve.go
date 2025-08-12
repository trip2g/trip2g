package handletgupdate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// var (
// 	// Global rate limiter: 10 requests per minute per user
// 	globalRateLimiter = NewRateLimiter(10, time.Minute)
// )

type Env interface {
	TgUserStateByBotIDAndChatID(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error)
	InsertTgUserProfile(ctx context.Context, arg db.InsertTgUserProfileParams) error
	UpsertTgUserState(ctx context.Context, arg db.UpsertTgUserStateParams) error
	UpsertTgBotChat(ctx context.Context, arg db.UpsertTgBotChatParams) error
	TgBotChatByTelegramID(ctx context.Context, telegramID int64) (db.TgBotChat, error)
	MarkTgBotChatRemoved(ctx context.Context, id int64) error
	UpdateTgBotChatCanInvite(ctx context.Context, arg db.UpdateTgBotChatCanInviteParams) error
	InsertTgChatMember(ctx context.Context, arg db.InsertTgChatMemberParams) error
	RemoveTgChatMember(ctx context.Context, arg db.RemoveTgChatMemberParams) error
	CalculateSha256(s string) string
	PublicURL() string
	LatestNoteViews() *model.NoteViews // TODO: read LiveNoteViews for production users
	Logger() logger.Logger
	BotID() int64
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	GetChatMemberStatus(ctx context.Context, chatID, userID int64) (string, error)
	GetBotCanInvite(ctx context.Context, chatID int64) (bool, error)
	BotLink() string

	GenerateTgAuthURL(ctx context.Context, path string, data model.TgAuthToken) (string, error)
	ListActiveTgChatSubgraphNamesByChatID(ctx context.Context, id int64) ([]string, error)
	InsertWaitListTgBotRequest(ctx context.Context, arg db.InsertWaitListTgBotRequestParams) error
	TgAttachCodeByCode(ctx context.Context, code string) (db.TgAttachCodeByCodeRow, error)
	DeleteTgAttachCode(ctx context.Context, code string) error
	UpdateUserTgID(ctx context.Context, arg db.UpdateUserTgIDParams) error
	ClearTgUserIDByTgUserID(ctx context.Context, tgUserID sql.NullInt64) error
	UserByTgUserID(ctx context.Context, tgUserID sql.NullInt64) (db.User, error)
	ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error)
	TgBotChatsWithSubgraphInvites(ctx context.Context, subgraphNames []string) ([]db.TgBotChatsWithSubgraphInvitesRow, error)
}

type UserStateData struct {
	QuizStates map[string]QuizState `json:"quiz_states"`
}

type UserState struct {
	*UserStateData

	ChatID int64
	Value  string

	UpdateCount int64
}

type request struct {
	chatID    int64
	update    tgbotapi.Update
	userState *UserState
	env       Env
	questions []Question
}

func Resolve(ctx context.Context, env Env, update tgbotapi.Update) error {
	// Rate limiting check
	// var userID int64
	// if update.Message != nil && update.Message.From != nil {
	// 	userID = update.Message.From.ID
	// } else if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
	// 	userID = update.CallbackQuery.From.ID
	// } else if update.MyChatMember != nil {
	// 	userID = update.MyChatMember.From.ID
	// } else if update.ChatMember != nil && update.ChatMember.From.ID != 0 {
	// 	userID = update.ChatMember.From.ID
	// }

	// if userID != 0 && !globalRateLimiter.Allow(userID) {
	// 	env.Logger().Error("Rate limit exceeded", "user_id", userID)
	// 	// Try to send rate limit message if possible
	// 	if update.Message != nil {
	// 		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⏰ Too many requests. Please wait before trying again.")
	// 		_, _ = env.Send(msg) // Ignore errors for rate limit messages
	// 	}
	// 	return nil // Don't return error to avoid processing issues
	// }

	// Update user profile if we have a message with user info
	if update.Message != nil && update.Message.From != nil {
		user := update.Message.From

		hash := env.CalculateSha256(fmt.Sprintf("%d%d%s%s%s",
			update.Message.Chat.ID, user.ID,
			user.FirstName, user.LastName, user.UserName,
		))

		err := env.InsertTgUserProfile(ctx, db.InsertTgUserProfileParams{
			ChatID:     update.Message.Chat.ID,
			BotID:      env.BotID(),
			Sha256Hash: hash,
			FirstName:  toNullString(user.FirstName),
			LastName:   toNullString(user.LastName),
			Username:   toNullString(user.UserName),
		})
		if err != nil {
			env.Logger().Error("failed to insert user profile", "error", err)
		}
	}

	chatID := int64(0)
	switch {
	case update.Message != nil:
		chatID = update.Message.Chat.ID
	case update.CallbackQuery != nil:
		chatID = update.CallbackQuery.Message.Chat.ID
	case update.MyChatMember != nil:
		chatID = update.MyChatMember.Chat.ID
	case update.ChatMember != nil:
		chatID = update.ChatMember.Chat.ID
	}

	req := &request{
		chatID: chatID,
		update: update,
		env:    env,
	}

	userState, err := req.UserState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user state: %w", err)
	}

	req.userState = userState

	defer func() {
		if req.userState != nil {
			updateErr := req.updateUserState(ctx)
			if updateErr != nil {
				env.Logger().Error("failed to update user state", "error", updateErr)
			}
		}
	}()

	// Handle bot being added/removed from chats
	if update.MyChatMember != nil {
		return req.handleMyChatMember(ctx)
	}

	// Handle users joining/leaving chats
	if update.ChatMember != nil {
		return req.handleChatMember(ctx)
	}

	if update.CallbackQuery != nil {
		return req.handleCallbackQuery(ctx)
	}

	if update.Message != nil && update.Message.IsCommand() {
		return req.handleCommands(ctx)
	}

	return nil
}

func (req *request) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(req.chatID, text)

	_, err := req.env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (req *request) SendCallbackMessage(text string) error {
	// Send callback response to close the loading state
	callback := tgbotapi.NewCallbackWithAlert(req.update.CallbackQuery.ID, text)

	_, err := req.env.Request(callback)
	if err != nil {
		return fmt.Errorf("failed to send callback message: %w", err)
	}

	return nil
}

func (req *request) handleCallbackQuery(ctx context.Context) error {
	log := req.env.Logger()

	// Validate callback data
	callbackData := req.update.CallbackQuery.Data
	if callbackData == "" {
		log.Error("Empty callback data received")
		return req.SendCallbackMessage("❌ Invalid request. Please try again.")
	}

	if len(callbackData) > 1000 {
		log.Error("Callback data too long", "length", len(callbackData))
		return req.SendCallbackMessage("❌ Request too large. Please try again.")
	}

	actionParts := strings.SplitN(callbackData, ":", 3)

	switch actionParts[0] {
	case "start_mbti":
		callback := tgbotapi.NewCallback(req.update.CallbackQuery.ID, req.update.CallbackQuery.Data)

		_, err := req.env.Request(callback)
		if err != nil {
			return fmt.Errorf("failed to send callback: %w", err)
		}

		return req.sendNextQuestion(ctx)

	case "mbti_answer":
		return req.handleMBTIAnswer(ctx, actionParts)

	default:
		log.Info("unhandled callback query", "data", req.update.CallbackQuery.Data)
	}

	return nil
}

func (req *request) handleCommands(ctx context.Context) error {
	switch req.update.Message.Command() {
	case "start":
		if req.update.Message.Chat.ID < 0 {
			return nil
		}

		args := req.update.Message.CommandArguments()
		if strings.HasPrefix(args, "group_") {
			return req.handleGroupAccess(ctx, args)
		}
		if strings.HasPrefix(args, "wl_") {
			return req.handleWaitListRequest(ctx, args)
		}
		if strings.HasPrefix(args, "attach_") {
			return req.handleAttachCode(ctx, args)
		}

		questions, err := req.Questions(ctx)
		if err != nil {
			return fmt.Errorf("failed to get questions: %w", err)
		}

		if len(questions) == 0 {
			return req.sendContentMenu(ctx)
		}

		return req.sendStartMenu(ctx)

	case "content":
		if req.update.Message.Chat.ID < 0 {
			url := fmt.Sprintf("%s?start=group_%d", req.env.BotLink(), req.update.Message.Chat.ID)
			msg := tgbotapi.NewMessage(req.update.Message.Chat.ID, "Доступ к материалам через бота")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Открыть", url),
				),
			)
			_, err := req.env.Send(msg)
			if err != nil {
				return fmt.Errorf("failed to send content message: %w", err)
			}

			return nil
		}

		return req.sendContentMenu(ctx)

	case "chats":
		return req.sendAvailableChats(ctx)

	default:
		msg := tgbotapi.NewMessage(req.update.Message.Chat.ID, "Unknown command")

		_, err := req.env.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func (req *request) UserState(ctx context.Context) (*UserState, error) {
	if req.userState != nil {
		return req.userState, nil
	}

	data, err := req.env.TgUserStateByBotIDAndChatID(ctx, db.TgUserStateByBotIDAndChatIDParams{
		BotID:  req.env.BotID(),
		ChatID: req.chatID,
	})
	if err != nil {
		// If not found, create a new one
		if errors.Is(err, sql.ErrNoRows) {
			req.userState = &UserState{
				UserStateData: &UserStateData{
					QuizStates: make(map[string]QuizState),
				},
				ChatID:      req.chatID,
				Value:       "pending",
				UpdateCount: 0,
			}
			return req.userState, nil
		}
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	var userStateData UserStateData
	err = json.Unmarshal([]byte(data.Data), &userStateData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state: %w", err)
	}

	if userStateData.QuizStates == nil {
		userStateData.QuizStates = make(map[string]QuizState)
	}

	return &UserState{
		UserStateData: &userStateData,
		ChatID:        data.ChatID,
		Value:         data.Value,
		UpdateCount:   data.UpdateCount,
	}, nil
}

func (req *request) updateUserState(ctx context.Context) error {
	if req.userState == nil {
		return nil
	}

	data, err := json.Marshal(req.userState.UserStateData)
	if err != nil {
		return fmt.Errorf("failed to marshal user state: %w", err)
	}

	params := db.UpsertTgUserStateParams{
		BotID:       req.env.BotID(),
		ChatID:      req.userState.ChatID,
		Value:       req.userState.Value,
		Data:        string(data),
		UpdateCount: req.userState.UpdateCount + 1,
	}

	err = req.env.UpsertTgUserState(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert user state: %w", err)
	}

	return nil
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func (req *request) handleWaitListRequest(ctx context.Context, args string) error {
	pathIDStr := strings.TrimPrefix(args, "wl_")
	pathID, err := strconv.ParseInt(pathIDStr, 10, 64)
	if err != nil {
		req.env.Logger().Warn("invalid path ID in wait list request", "args", args, "pathID", pathIDStr, "error", err)
		return req.SendMessage("❌ Invalid request. Please try again.")
	}

	err = req.env.InsertWaitListTgBotRequest(ctx, db.InsertWaitListTgBotRequestParams{
		BotID:      req.env.BotID(),
		ChatID:     req.update.Message.Chat.ID,
		NotePathID: pathID,
	})
	if err != nil {
		req.env.Logger().Error("failed to insert wait list request", "error", err, "pathID", pathID, "chatID", req.update.Message.Chat.ID)
		return req.SendMessage("❌ Failed to process your request. Please try again later.")
	}

	return req.SendMessage("✅ Thank you for your interest! You've been added to the wait list. We'll notify you when access becomes available.")
}

func (req *request) handleAttachCode(ctx context.Context, args string) error {
	// Extract the code from "attach_XXXXXXXX"
	if len(args) < 8 || !strings.HasPrefix(args, "attach_") {
		return req.SendMessage("❌ Неверный формат кода привязки.")
	}

	code := strings.TrimPrefix(args, "attach_")
	if len(code) != 8 {
		return req.SendMessage("❌ Неверный формат кода привязки.")
	}

	// Get the attach code from database
	attachCode, err := req.env.TgAttachCodeByCode(ctx, code)
	if err != nil {
		if db.IsNoFound(err) {
			return req.SendMessage("❌ Код привязки не найден или истёк.")
		}
		req.env.Logger().Error("failed to get attach code", "error", err, "code", code)
		return req.SendMessage("❌ Не удалось обработать код привязки. Попробуйте снова.")
	}

	// Verify the bot ID matches
	if attachCode.BotID != req.env.BotID() {
		return req.SendMessage("❌ Этот код привязки предназначен для другого бота.")
	}

	// Get the telegram user ID from the message
	if req.update.Message.From == nil {
		return req.SendMessage("❌ Не удалось определить ваш Telegram аккаунт.")
	}

	telegramUserID := req.update.Message.From.ID

	// If the account already has a different Telegram user attached, notify the old user
	if attachCode.CurrentTgUserID.Valid && attachCode.CurrentTgUserID.Int64 != telegramUserID {
		// Send notification to the old Telegram user
		oldUserMsg := tgbotapi.NewMessage(attachCode.CurrentTgUserID.Int64, 
			"⚠️ Ваш аккаунт был привязан к другому Telegram пользователю.\n\n" +
			"Если это были не вы, обратитесь в поддержку.")
		// Try to send but don't fail if we can't reach the old user
		_, _ = req.env.Send(oldUserMsg)
	}

	// Clear this Telegram ID from any other users first
	err = req.env.ClearTgUserIDByTgUserID(ctx, sql.NullInt64{Int64: telegramUserID, Valid: true})
	if err != nil {
		req.env.Logger().Error("failed to clear telegram ID from other users", "error", err, "telegramUserID", telegramUserID)
		return req.SendMessage("❌ Не удалось очистить предыдущие привязки. Попробуйте снова.")
	}

	// Update the user's telegram ID
	err = req.env.UpdateUserTgID(ctx, db.UpdateUserTgIDParams{
		TgUserID: sql.NullInt64{Int64: telegramUserID, Valid: true},
		ID:       attachCode.UserID,
	})
	if err != nil {
		req.env.Logger().Error("failed to update user telegram ID", "error", err, "userID", attachCode.UserID, "telegramUserID", telegramUserID)
		return req.SendMessage("❌ Не удалось привязать аккаунт. Попробуйте снова.")
	}

	// Delete the used attach code
	err = req.env.DeleteTgAttachCode(ctx, code)
	if err != nil {
		req.env.Logger().Error("failed to delete attach code", "error", err, "code", code)
		// Don't return error here as the main operation succeeded
	}

	return req.SendMessage("✅ Ваш Telegram аккаунт успешно привязан! Теперь вы можете получить доступ к контенту через этого бота.")
}

func (req *request) sendAvailableChats(ctx context.Context) error {
	// Check if user has Telegram ID linked
	if req.update.Message.From == nil {
		return req.SendMessage("❌ Не удалось определить ваш Telegram аккаунт.")
	}

	// Get user by Telegram ID
	user, err := req.env.UserByTgUserID(ctx, sql.NullInt64{Int64: req.update.Message.From.ID, Valid: true})
	if err != nil {
		if db.IsNoFound(err) {
			return req.SendMessage("❌ Ваш Telegram аккаунт не привязан. Используйте команду в веб-интерфейсе для привязки.")
		}
		req.env.Logger().Error("failed to get user by telegram ID", "error", err, "telegramID", req.update.Message.From.ID)
		return req.SendMessage("❌ Произошла ошибка. Попробуйте позже.")
	}

	// Get user's active subgraphs
	activeSubgraphs, err := req.env.ListActiveUserSubgraphs(ctx, user.ID)
	if err != nil {
		req.env.Logger().Error("failed to get active user subgraphs", "error", err, "userID", user.ID)
		return req.SendMessage("❌ Не удалось получить список доступных групп.")
	}

	if len(activeSubgraphs) == 0 {
		return req.SendMessage("У вас нет активных подписок с доступом к групповым чатам.")
	}

	// Get chats with invites for user's subgraphs
	chats, err := req.env.TgBotChatsWithSubgraphInvites(ctx, activeSubgraphs)
	if err != nil {
		req.env.Logger().Error("failed to get chats with invites", "error", err, "subgraphs", activeSubgraphs)
		return req.SendMessage("❌ Не удалось получить список чатов.")
	}

	if len(chats) == 0 {
		return req.SendMessage("Нет доступных групповых чатов для ваших подписок.")
	}

	// Build message with chat links
	var message strings.Builder
	message.WriteString("📋 *Доступные групповые чаты:*\n\n")

	// Group chats by subgraph
	chatsBySubgraph := make(map[string][]db.TgBotChatsWithSubgraphInvitesRow)
	for _, chat := range chats {
		chatsBySubgraph[chat.SubgraphName] = append(chatsBySubgraph[chat.SubgraphName], chat)
	}

	// Build keyboard with chat links
	var rows [][]tgbotapi.InlineKeyboardButton
	for subgraph, subgraphChats := range chatsBySubgraph {
		message.WriteString(fmt.Sprintf("*%s:*\n", subgraph))
		for _, chat := range subgraphChats {
			message.WriteString(fmt.Sprintf("• %s\n", chat.ChatTitle))
			
			// Create invite link
			// For supergroups/channels, Telegram ID is negative and we need to remove the -100 prefix
			chatID := chat.TelegramID
			if chatID < -1000000000000 {
				chatID = -chatID - 1000000000000
			} else if chatID < 0 {
				chatID = -chatID
			}
			inviteLink := fmt.Sprintf("https://t.me/c/%d", chatID)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(chat.ChatTitle, inviteLink),
			))
		}
		message.WriteString("\n")
	}

	msg := tgbotapi.NewMessage(req.chatID, message.String())
	msg.ParseMode = "Markdown"
	if len(rows) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	}

	_, err = req.env.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send chats message: %w", err)
	}

	return nil
}
