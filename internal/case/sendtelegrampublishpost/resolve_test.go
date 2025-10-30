package sendtelegrampublishpost_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/sendtelegrampublishpost"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestResolve(t *testing.T) {
	// Create a proper AST for the test content - use simple paragraph text instead of heading to avoid warnings
	content := []byte("Test note content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

	// Create a test note view with proper AST
	testNote := &model.NoteView{
		Path:      "/test-note",
		Title:     "Test Note",
		PathID:    123,
		VersionID: 456,
		Content:   content,
		HTML:      "<p>Test note content</p>",
		Permalink: "/test-note",
		Free:      true,
		RawMeta: map[string]interface{}{
			"title": "Test Note",
			"free":  true,
		},
		InLinks:       map[string]struct{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]*model.NoteAssetReplace{},
	}

	// Set the AST using the SetAst method
	testNote.SetAst(doc)

	// Create NoteViews containing the test note
	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-note": testNote,
		},
		List:      []*model.NoteView{testNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "latest",
	}

	mockChats := []db.TgBotChat{
		{
			ID:         1,
			TelegramID: -1001234567890,
			ChatType:   "supergroup",
			ChatTitle:  "Test Chat 1",
			BotID:      1,
		},
		{
			ID:         2,
			TelegramID: -1001234567891,
			ChatType:   "supergroup",
			ChatTitle:  "Test Chat 2",
			BotID:      1,
		},
	}

	tests := []struct {
		name     string
		noteID   int64
		instant  bool
		setupEnv func() *EnvMock
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "success - regular post to multiple chats",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						require.Equal(t, int64(123), notePathID)
						return mockChats, nil
					},
					SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
						require.Contains(t, []int64{1, 2}, chatID)
						telegramMsg, ok := msg.(tgbotapi.MessageConfig)
						require.True(t, ok)
						require.Equal(t, "HTML", telegramMsg.ParseMode)
						require.Contains(t, telegramMsg.Text, "Test note content")
						return 100 + chatID, nil // Return unique message IDs
					},
					InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
						require.Equal(t, int64(123), arg.NotePathID)
						require.Contains(t, []int64{1, 2}, arg.ChatID)
						require.Contains(t, []int64{101, 102}, arg.MessageID) // Expected message IDs
						require.Equal(t, int64(0), arg.Instant)               // Regular posts are not instant
						return nil
					},
					UpdateTelegramPublishNoteAsPublishedFunc: func(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error {
						require.Equal(t, int64(123), arg.NotePathID)
						require.True(t, arg.PublishedVersionID.Valid)
						require.Equal(t, int64(456), arg.PublishedVersionID.Int64)
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:    "success - instant post to multiple chats",
			noteID:  123,
			instant: true,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotInstantChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						require.Equal(t, int64(123), notePathID)
						return mockChats, nil
					},
					SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
						require.Contains(t, []int64{1, 2}, chatID)
						return 100 + chatID, nil
					},
					// Instant posts should now save sent messages
					InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
						require.Equal(t, int64(123), arg.NotePathID)
						require.Contains(t, []int64{1, 2}, arg.ChatID)
						require.Equal(t, int64(1), arg.Instant) // Instant posts have instant=1
						return nil
					},
					// But should not mark note as published
					UpdateTelegramPublishNoteAsPublishedFunc: func(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error {
						t.Error("UpdateTelegramPublishNoteAsPublished should not be called for instant posts")
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:    "success - instant post with no chats (should not error)",
			noteID:  123,
			instant: true,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotInstantChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						return []db.TgBotChat{}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:    "error - note view not found",
			noteID:  999, // Non-existent note ID
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
				}
			},
			wantErr: true,
			errMsg:  "note view not found for path ID 999",
		},
		{
			name:    "error - regular post with no chats",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						return []db.TgBotChat{}, nil
					},
				}
			},
			wantErr: true,
			errMsg:  "no chat IDs found for note path ID 123",
		},
		{
			name:    "error - failed to get chats",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						return nil, errors.New("database error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to get chat IDs for note: database error",
		},
		{
			name:    "error - failed to send telegram message",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						return []db.TgBotChat{mockChats[0]}, nil // Only one chat
					},
					SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
						return 0, errors.New("telegram API error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to send telegram message to chat 1: telegram API error",
		},
		{
			name:    "error - failed to record sent message",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						return []db.TgBotChat{mockChats[0]}, nil
					},
					SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
						return 101, nil
					},
					InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
						return errors.New("database insert error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to record sent message for chat 1: database insert error",
		},
		{
			name:    "error - failed to mark note as published",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTgBotChatsByTelegramPublishNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.TgBotChat, error) {
						return []db.TgBotChat{mockChats[0]}, nil
					},
					SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
						return 101, nil
					},
					InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
						return nil
					},
					UpdateTelegramPublishNoteAsPublishedFunc: func(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error {
						return errors.New("database update error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to mark note as published: database update error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			ctx := context.Background()

			err := sendtelegrampublishpost.Resolve(ctx, env, tt.noteID, tt.instant)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetByPathID(t *testing.T) {
	// Test the GetByPathID method from NoteViews
	testNote := &model.NoteView{
		PathID: 123,
		Path:   "/test-note",
		Title:  "Test Note",
	}

	noteViews := &model.NoteViews{
		Map:  map[string]*model.NoteView{"/test-note": testNote}, // GetByPathID iterates over Map, not List
		List: []*model.NoteView{testNote},
	}

	// Test finding existing note
	found := noteViews.GetByPathID(123)
	require.NotNil(t, found)
	require.Equal(t, int64(123), found.PathID)
	require.Equal(t, "Test Note", found.Title)

	// Test finding non-existent note
	notFound := noteViews.GetByPathID(999)
	require.Nil(t, notFound)
}
