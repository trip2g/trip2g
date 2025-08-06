//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

package handletgupdate

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		update  tgbotapi.Update
		setup   func(*EnvMock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "start command with group access",
			update: tgbotapi.Update{
				UpdateID: 430606726,
				Message: &tgbotapi.Message{
					MessageID: 361,
					From: &tgbotapi.User{
						ID:           7828312136,
						IsBot:        false,
						FirstName:    "Алексей",
						LanguageCode: "en",
					},
					Chat: &tgbotapi.Chat{
						ID:        7828312136,
						FirstName: "Алексей",
						Type:      "private",
					},
					Date: 1750942673,
					Text: "/start group_-1002529281698",
					Entities: []tgbotapi.MessageEntity{
						{
							Type:   "bot_command",
							Offset: 0,
							Length: 6,
						},
					},
				},
			},
			setup: func(env *EnvMock) {
				// User profile insertion
				env.CalculateSha256Func = func(s string) string {
					return "test_hash_123"
				}
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.InsertTgUserProfileFunc = func(ctx context.Context, arg db.InsertTgUserProfileParams) error {
					return nil
				}

				// User state - not found initially (new user)
				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{}, sql.ErrNoRows
				}

				// Chat member insertion for group access
				env.InsertTgChatMemberFunc = func(ctx context.Context, arg db.InsertTgChatMemberParams) error {
					return nil
				}

				// Message sending
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}

				// User state upsert
				env.UpsertTgUserStateFunc = func(ctx context.Context, arg db.UpsertTgUserStateParams) error {
					return nil
				}

				// Logger
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}

				// GenerateTgAuthURL for group access
				env.GenerateTgAuthURLFunc = func(ctx context.Context, path string, data model.TgAuthToken) (string, error) {
					return "https://example.com/auth?token=test", nil
				}

				// GetChatMemberStatus for group membership verification
				env.GetChatMemberStatusFunc = func(ctx context.Context, chatID int64, userID int64) (string, error) {
					// Mock successful group membership check
					return "member", nil
				}

				// ListActiveTgChatSubgraphNamesByChatID for content menu
				env.ListActiveTgChatSubgraphNamesByChatIDFunc = func(ctx context.Context, id int64) ([]string, error) {
					return []string{"test-subgraph"}, nil
				}
				env.LatestNoteViewsFunc = func() *model.NoteViews {
					return &model.NoteViews{
						Subgraphs: map[string]*model.NoteSubgraph{
							"test-subgraph": {},
						},
					}
				}
			},
			wantErr: false,
		},
		{
			name: "regular start command",
			update: tgbotapi.Update{
				UpdateID: 430606727,
				Message: &tgbotapi.Message{
					MessageID: 362,
					From: &tgbotapi.User{
						ID:           7828312136,
						IsBot:        false,
						FirstName:    "Алексей",
						LanguageCode: "en",
					},
					Chat: &tgbotapi.Chat{
						ID:        7828312136,
						FirstName: "Алексей",
						Type:      "private",
					},
					Date: 1750942700,
					Text: "/start",
					Entities: []tgbotapi.MessageEntity{
						{
							Type:   "bot_command",
							Offset: 0,
							Length: 6,
						},
					},
				},
			},
			setup: func(env *EnvMock) {
				env.CalculateSha256Func = func(s string) string {
					return "test_hash_123"
				}
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.InsertTgUserProfileFunc = func(ctx context.Context, arg db.InsertTgUserProfileParams) error {
					return nil
				}
				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{}, sql.ErrNoRows
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
				env.UpsertTgUserStateFunc = func(ctx context.Context, arg db.UpsertTgUserStateParams) error {
					return nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.PublicURLFunc = func() string {
					return "https://test.com"
				}
				env.LatestNoteViewsFunc = func() *model.NoteViews {
					return &model.NoteViews{
						List: []*model.NoteView{
							{Title: "Test Note", Path: "/test"},
						},
					}
				}
				env.ListActiveTgChatSubgraphNamesByChatIDFunc = func(ctx context.Context, id int64) ([]string, error) {
					return []string{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "existing user with valid state",
			update: tgbotapi.Update{
				UpdateID: 430606728,
				Message: &tgbotapi.Message{
					MessageID: 363,
					From: &tgbotapi.User{
						ID:           7828312136,
						IsBot:        false,
						FirstName:    "Алексей",
						LanguageCode: "en",
					},
					Chat: &tgbotapi.Chat{
						ID:        7828312136,
						FirstName: "Алексей",
						Type:      "private",
					},
					Date: 1750942800,
					Text: "/start",
				},
			},
			setup: func(env *EnvMock) {
				env.CalculateSha256Func = func(s string) string {
					return "test_hash_123"
				}
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.InsertTgUserProfileFunc = func(ctx context.Context, arg db.InsertTgUserProfileParams) error {
					return nil
				}

				// Return existing user state with valid JSON
				validStateData := UserStateData{
					QuizStates: map[string]QuizState{
						"mbti": {Answers: map[int]int{}},
					},
				}
				stateJSON, _ := json.Marshal(validStateData)

				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{
						ChatID:      7828312136,
						BotID:       1,
						Value:       "pending",
						Data:        string(stateJSON),
						UpdateCount: 0,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}, nil
				}

				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
				env.UpsertTgUserStateFunc = func(ctx context.Context, arg db.UpsertTgUserStateParams) error {
					return nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.PublicURLFunc = func() string {
					return "https://test.com"
				}
				env.LatestNoteViewsFunc = func() *model.NoteViews {
					return &model.NoteViews{List: []*model.NoteView{}}
				}
			},
			wantErr: false,
		},
		{
			name: "user profile insertion error",
			update: tgbotapi.Update{
				UpdateID: 430606729,
				Message: &tgbotapi.Message{
					MessageID: 364,
					From: &tgbotapi.User{
						ID:        7828312136,
						FirstName: "Алексей",
					},
					Chat: &tgbotapi.Chat{
						ID:   7828312136,
						Type: "private",
					},
					Date: 1750942900,
					Text: "/start",
				},
			},
			setup: func(env *EnvMock) {
				env.CalculateSha256Func = func(s string) string {
					return "test_hash_123"
				}
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.InsertTgUserProfileFunc = func(ctx context.Context, arg db.InsertTgUserProfileParams) error {
					return sql.ErrConnDone // Simulate database error
				}
				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{}, sql.ErrNoRows
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
				env.UpsertTgUserStateFunc = func(ctx context.Context, arg db.UpsertTgUserStateParams) error {
					return nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.PublicURLFunc = func() string {
					return "https://test.com"
				}
				env.LatestNoteViewsFunc = func() *model.NoteViews {
					return &model.NoteViews{List: []*model.NoteView{}}
				}
			},
			wantErr: false, // Error is logged but doesn't fail the request
		},
		{
			name: "invalid user state JSON",
			update: tgbotapi.Update{
				UpdateID: 430606730,
				Message: &tgbotapi.Message{
					MessageID: 365,
					From: &tgbotapi.User{
						ID:        7828312136,
						FirstName: "Алексей",
					},
					Chat: &tgbotapi.Chat{
						ID:   7828312136,
						Type: "private",
					},
					Date: 1750943000,
					Text: "/start",
				},
			},
			setup: func(env *EnvMock) {
				env.CalculateSha256Func = func(s string) string {
					return "test_hash_123"
				}
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.InsertTgUserProfileFunc = func(ctx context.Context, arg db.InsertTgUserProfileParams) error {
					return nil
				}

				// Return existing user state with invalid JSON (like "pending")
				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{
						ChatID:      7828312136,
						BotID:       1,
						Value:       "pending",
						Data:        "invalid json data", // Invalid JSON
						UpdateCount: 0,
					}, nil
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			wantErr: true,
			errMsg:  "failed to get user state: failed to unmarshal user state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setup(env)

			err := Resolve(context.Background(), env, tt.update)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify expected calls were made
			if tt.update.Message != nil && tt.update.Message.From != nil {
				require.Len(t, env.InsertTgUserProfileCalls(), 1)
				require.Len(t, env.TgUserStateByBotIDAndChatIDCalls(), 1)
			}
		})
	}
}

func TestUserState(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*EnvMock)
		chatID        int64
		expectedValue string
		wantErr       bool
		errMsg        string
	}{
		{
			name: "new user state creation",
			setup: func(env *EnvMock) {
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{}, sql.ErrNoRows
				}
			},
			chatID:        123456,
			expectedValue: "pending",
			wantErr:       false,
		},
		{
			name: "existing user state",
			setup: func(env *EnvMock) {
				env.BotIDFunc = func() int64 {
					return 1
				}
				validStateData := UserStateData{
					QuizStates: map[string]QuizState{
						"mbti": {Answers: map[int]int{0: 1, 1: 2}},
					},
				}
				stateJSON, _ := json.Marshal(validStateData)

				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{
						ChatID:      123456,
						BotID:       1,
						Value:       "quiz_in_progress",
						Data:        string(stateJSON),
						UpdateCount: 3,
					}, nil
				}
			},
			chatID:        123456,
			expectedValue: "quiz_in_progress",
			wantErr:       false,
		},
		{
			name: "database error",
			setup: func(env *EnvMock) {
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.TgUserStateByBotIDAndChatIDFunc = func(ctx context.Context, arg db.TgUserStateByBotIDAndChatIDParams) (db.TgUserState, error) {
					return db.TgUserState{}, sql.ErrConnDone
				}
			},
			chatID:  123456,
			wantErr: true,
			errMsg:  "failed to get user state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setup(env)

			req := &request{
				chatID: tt.chatID,
				env:    env,
			}

			userState, err := req.UserState(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, userState)
				require.Equal(t, tt.expectedValue, userState.Value)
				require.Equal(t, tt.chatID, userState.ChatID)
				require.NotNil(t, userState.UserStateData)
				require.NotNil(t, userState.QuizStates)
			}
		})
	}
}

func TestHandleWaitListRequest(t *testing.T) {
	tests := []struct {
		name        string
		update      tgbotapi.Update
		setupEnv    func(*EnvMock)
		wantErr     bool
		wantMessage string
	}{
		{
			name: "successful wait list request",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: 1,
					From: &tgbotapi.User{
						ID:        123456789,
						FirstName: "Test",
						UserName:  "testuser",
					},
					Chat: &tgbotapi.Chat{
						ID:   123456789,
						Type: "private",
					},
					Date: 1750942700,
					Text: "/start wl_42",
					Entities: []tgbotapi.MessageEntity{
						{
							Type:   "bot_command",
							Offset: 0,
							Length: 6,
						},
					},
				},
			},
			setupEnv: func(env *EnvMock) {
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.BotIDFunc = func() int64 {
					return 1
				}
				env.InsertWaitListTgBotRequestFunc = func(ctx context.Context, arg db.InsertWaitListTgBotRequestParams) error {
					require.Equal(t, int64(1), arg.BotID)
					require.Equal(t, int64(123456789), arg.ChatID)
					require.Equal(t, int64(42), arg.NotePathID)
					return nil
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "invalid path ID format",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: 1,
					From: &tgbotapi.User{
						ID:        123456789,
						FirstName: "Test",
						UserName:  "testuser",
					},
					Chat: &tgbotapi.Chat{
						ID:   123456789,
						Type: "private",
					},
					Date: 1750942700,
					Text: "/start wl_invalid",
					Entities: []tgbotapi.MessageEntity{
						{
							Type:   "bot_command",
							Offset: 0,
							Length: 6,
						},
					},
				},
			},
			setupEnv: func(env *EnvMock) {
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			if tt.setupEnv != nil {
				tt.setupEnv(env)
			}

			req := &request{
				chatID: tt.update.Message.Chat.ID,
				update: tt.update,
				env:    env,
			}

			err := req.handleWaitListRequest(context.Background(), tt.update.Message.CommandArguments())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
