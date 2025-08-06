package handletgupdate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

func TestHandleGroupAccess(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		update  tgbotapi.Update
		setup   func(*EnvMock)
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid group access",
			args: "group_-1002529281698",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					From: &tgbotapi.User{
						ID:        7828312136,
						FirstName: "Алексей",
					},
					Chat: &tgbotapi.Chat{
						ID:   7828312136,
						Type: "private",
					},
				},
			},
			setup: func(env *EnvMock) {
				env.GetChatMemberStatusFunc = func(ctx context.Context, chatID, userID int64) (string, error) {
					// Mock successful group membership check
					return "member", nil
				}
				env.InsertTgChatMemberFunc = func(ctx context.Context, arg db.InsertTgChatMemberParams) error {
					// Verify correct parameters
					expectedUserID := int64(7828312136)
					expectedChatID := int64(-1002529281698)
					require.Equal(t, expectedUserID, arg.UserID)
					require.Equal(t, expectedChatID, arg.ChatID)
					return nil
				}
				env.BotIDFunc = func() int64 {
					return 987654321
				}
				env.GenerateTgAuthURLFunc = func(ctx context.Context, path string, data model.TgAuthToken) (string, error) {
					return "https://example.com/auth?token=test", nil
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
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
			name: "invalid group ID format",
			args: "group_invalid",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					From: &tgbotapi.User{
						ID: 7828312136,
					},
					Chat: &tgbotapi.Chat{
						ID: 7828312136,
					},
				},
			},
			setup: func(env *EnvMock) {
				// SendMessage should be called for error message
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
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
			wantErr: false, // Function returns error from SendMessage, not parsing error
		},
		{
			name: "database error on insert",
			args: "group_-1002529281698",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					From: &tgbotapi.User{
						ID: 7828312136,
					},
					Chat: &tgbotapi.Chat{
						ID: 7828312136,
					},
				},
			},
			setup: func(env *EnvMock) {
				env.GetChatMemberStatusFunc = func(ctx context.Context, chatID, userID int64) (string, error) {
					// Mock successful group membership check
					return "member", nil
				}
				env.InsertTgChatMemberFunc = func(ctx context.Context, arg db.InsertTgChatMemberParams) error {
					return sql.ErrConnDone // Database error
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.BotIDFunc = func() int64 {
					return 987654321
				}
				env.GenerateTgAuthURLFunc = func(ctx context.Context, path string, data model.TgAuthToken) (string, error) {
					return "https://example.com/auth?token=test", nil
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
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
			wantErr: false, // Error is logged but function returns SendMessage result
		},
		{
			name: "user not member of group",
			args: "group_-1002529281698",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					From: &tgbotapi.User{
						ID: 7828312136,
					},
					Chat: &tgbotapi.Chat{
						ID: 7828312136,
					},
				},
			},
			setup: func(env *EnvMock) {
				env.GetChatMemberStatusFunc = func(ctx context.Context, chatID, userID int64) (string, error) {
					// Mock user not being a member
					return "", errors.New("telegram API error: Bad Request: user not found")
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
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
			wantErr: false, // Function returns SendMessage result, not the verification error
		},
		{
			name: "user has invalid status (left)",
			args: "group_-1002529281698",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					From: &tgbotapi.User{
						ID: 7828312136,
					},
					Chat: &tgbotapi.Chat{
						ID: 7828312136,
					},
				},
			},
			setup: func(env *EnvMock) {
				env.GetChatMemberStatusFunc = func(ctx context.Context, chatID, userID int64) (string, error) {
					// Mock user with "left" status
					return "left", nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				env.SendFunc = func(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
					return tgbotapi.Message{}, nil
				}
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
			wantErr: false, // Function returns SendMessage result, not the verification error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setup(env)

			req := &request{
				update: tt.update,
				env:    env,
			}

			err := req.handleGroupAccess(context.Background(), tt.args)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify expected calls
			if tt.args != "group_invalid" && tt.name != "user not member of group" && tt.name != "user has invalid status (left)" {
				require.Len(t, env.InsertTgChatMemberCalls(), 1)
			}
			require.Len(t, env.SendCalls(), 1)
		})
	}
}

func TestHandleMyChatMember(t *testing.T) {
	tests := []struct {
		name   string
		update tgbotapi.Update
		setup  func(*EnvMock)
	}{
		{
			name: "bot added to group",
			update: tgbotapi.Update{
				MyChatMember: &tgbotapi.ChatMemberUpdated{
					Chat: tgbotapi.Chat{
						ID:    -1002529281698,
						Type:  "supergroup",
						Title: "Test Group",
					},
					OldChatMember: tgbotapi.ChatMember{
						Status: "left",
					},
					NewChatMember: tgbotapi.ChatMember{
						Status: "member",
					},
				},
			},
			setup: func(env *EnvMock) {
				env.UpsertTgBotChatFunc = func(ctx context.Context, arg db.UpsertTgBotChatParams) error {
					require.Equal(t, int64(-1002529281698), arg.ID)
					require.Equal(t, "supergroup", arg.ChatType)
					require.Equal(t, "Test Group", arg.ChatTitle)
					return nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
		},
		{
			name: "bot removed from group",
			update: tgbotapi.Update{
				MyChatMember: &tgbotapi.ChatMemberUpdated{
					Chat: tgbotapi.Chat{
						ID:   -1002529281698,
						Type: "supergroup",
					},
					OldChatMember: tgbotapi.ChatMember{
						Status: "member",
					},
					NewChatMember: tgbotapi.ChatMember{
						Status: "left",
					},
				},
			},
			setup: func(env *EnvMock) {
				env.MarkTgBotChatRemovedFunc = func(ctx context.Context, id int64) error {
					require.Equal(t, int64(-1002529281698), id)
					return nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
		},
		{
			name: "private chat - should not track",
			update: tgbotapi.Update{
				MyChatMember: &tgbotapi.ChatMemberUpdated{
					Chat: tgbotapi.Chat{
						ID:   7828312136,
						Type: "private",
					},
					OldChatMember: tgbotapi.ChatMember{
						Status: "member",
					},
					NewChatMember: tgbotapi.ChatMember{
						Status: "member",
					},
				},
			},
			setup: func(env *EnvMock) {
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
				// No database calls should be made for private chats
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setup(env)

			req := &request{
				update: tt.update,
				env:    env,
			}

			err := req.handleMyChatMember(context.Background())
			require.NoError(t, err)

			// Verify expected calls based on chat type and status change
			if tt.update.MyChatMember.Chat.Type != "private" {
				oldStatus := tt.update.MyChatMember.OldChatMember.Status
				newStatus := tt.update.MyChatMember.NewChatMember.Status

				if (newStatus == "member" || newStatus == "administrator") &&
					(oldStatus == "left" || oldStatus == "kicked") {
					require.Len(t, env.UpsertTgBotChatCalls(), 1)
				}

				if (newStatus == "left" || newStatus == "kicked") &&
					(oldStatus == "member" || oldStatus == "administrator") {
					require.Len(t, env.MarkTgBotChatRemovedCalls(), 1)
				}
			}
		})
	}
}

func TestHandleChatMember(t *testing.T) {
	tests := []struct {
		name   string
		update tgbotapi.Update
		setup  func(*EnvMock)
	}{
		{
			name: "user joined group",
			update: tgbotapi.Update{
				ChatMember: &tgbotapi.ChatMemberUpdated{
					Chat: tgbotapi.Chat{
						ID:   -1002529281698,
						Type: "supergroup",
					},
					From: tgbotapi.User{
						ID: 7828312136,
					},
					OldChatMember: tgbotapi.ChatMember{
						Status: "left",
						User: &tgbotapi.User{
							ID: 7828312136,
						},
					},
					NewChatMember: tgbotapi.ChatMember{
						Status: "member",
						User: &tgbotapi.User{
							ID: 7828312136,
						},
					},
				},
			},
			setup: func(env *EnvMock) {
				env.RequestFunc = func(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
					// Mock successful group membership check
					chatMember := tgbotapi.ChatMember{
						Status: "member",
						User: &tgbotapi.User{
							ID: 7828312136,
						},
					}
					memberJSON, _ := json.Marshal(chatMember)
					return &tgbotapi.APIResponse{
						Ok:     true,
						Result: memberJSON,
					}, nil
				}
				env.InsertTgChatMemberFunc = func(ctx context.Context, arg db.InsertTgChatMemberParams) error {
					require.Equal(t, int64(7828312136), arg.UserID)
					require.Equal(t, int64(-1002529281698), arg.ChatID)
					return nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
		},
		{
			name: "user left group",
			update: tgbotapi.Update{
				ChatMember: &tgbotapi.ChatMemberUpdated{
					Chat: tgbotapi.Chat{
						ID:   -1002529281698,
						Type: "supergroup",
					},
					From: tgbotapi.User{
						ID: 7828312136,
					},
					OldChatMember: tgbotapi.ChatMember{
						Status: "member",
						User: &tgbotapi.User{
							ID: 7828312136,
						},
					},
					NewChatMember: tgbotapi.ChatMember{
						Status: "left",
						User: &tgbotapi.User{
							ID: 7828312136,
						},
					},
				},
			},
			setup: func(env *EnvMock) {
				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					require.Equal(t, int64(7828312136), arg.UserID)
					require.Equal(t, int64(-1002529281698), arg.ChatID)
					return nil
				}
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
		},
		{
			name: "private chat - should not track",
			update: tgbotapi.Update{
				ChatMember: &tgbotapi.ChatMemberUpdated{
					Chat: tgbotapi.Chat{
						ID:   7828312136,
						Type: "private",
					},
				},
			},
			setup: func(env *EnvMock) {
				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setup(env)

			req := &request{
				update: tt.update,
				env:    env,
			}

			err := req.handleChatMember(context.Background())
			require.NoError(t, err)

			// Verify expected calls based on chat type and status change
			if tt.update.ChatMember != nil && tt.update.ChatMember.Chat.Type != "private" {
				oldStatus := tt.update.ChatMember.OldChatMember.Status
				newStatus := tt.update.ChatMember.NewChatMember.Status

				if (newStatus == "member" || newStatus == "administrator" || newStatus == "creator") &&
					(oldStatus == "left" || oldStatus == "kicked" || oldStatus == "") {
					require.Len(t, env.InsertTgChatMemberCalls(), 1)
				}

				if (newStatus == "left" || newStatus == "kicked") &&
					(oldStatus == "member" || oldStatus == "administrator" || oldStatus == "creator") {
					require.Len(t, env.RemoveTgChatMemberCalls(), 1)
				}
			}
		})
	}
}
