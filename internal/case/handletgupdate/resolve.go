package handletgupdate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

	GenerateTgAuthURL(ctx context.Context, path string, data model.TgAuthToken) (string, error)
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

func (req *request) handleCallbackQuery(ctx context.Context) error {
	log := req.env.Logger()

	actionParts := strings.SplitN(req.update.CallbackQuery.Data, ":", 3)

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
