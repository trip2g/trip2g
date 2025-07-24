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
	MarkTgBotChatRemoved(ctx context.Context, id int64) error
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

	GenerateTgAuthURL(ctx context.Context, path string, data model.TgAuthToken) (string, error)
	ListActiveTgChatSubgraphNamesByChatID(ctx context.Context, id sql.NullInt64) ([]string, error)
	InsertWaitListTgBotRequest(ctx context.Context, arg db.InsertWaitListTgBotRequestParams) error
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
		hash := env.CalculateSha256(fmt.Sprintf("%s%s%s", user.FirstName, user.LastName, user.UserName))

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
		args := req.update.Message.CommandArguments()
		if strings.HasPrefix(args, "group_") {
			return req.handleGroupAccess(ctx, args)
		}
		if strings.HasPrefix(args, "wl_") {
			return req.handleWaitListRequest(ctx, args)
		}
		return req.sendStartMenu(ctx)

	case "content":
		return req.sendContentMenu(ctx)

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

	err = req.env.UpsertTgUserState(ctx, db.UpsertTgUserStateParams{
		BotID:       req.env.BotID(),
		ChatID:      req.userState.ChatID,
		Value:       req.userState.Value,
		Data:        string(data),
		UpdateCount: req.userState.UpdateCount + 1,
	})
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
