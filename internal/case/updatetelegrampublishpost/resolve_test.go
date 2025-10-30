package updatetelegrampublishpost_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/updatetelegrampublishpost"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestResolve(t *testing.T) {
	// Create a proper AST for the test content
	content := []byte("Updated test note content")
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
		HTML:      "<p>Updated test note content</p>",
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

	mockSentMessages := []db.ListTelegramPublishSentMessagesByNotePathIDRow{
		{
			ChatID:      1,
			MessageID:   101,
			TelegramID:  -1001234567890,
			ContentHash: "oldhash1",
		},
		{
			ChatID:      2,
			MessageID:   102,
			TelegramID:  -1001234567891,
			ContentHash: "oldhash2",
		},
	}

	tests := []struct {
		name     string
		noteID   int64
		setupEnv func() *EnvMock
		wantErr  bool
		errMsg   string
	}{
		{
			name:   "success - update multiple messages",
			noteID: 123,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Updated test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						require.Equal(t, int64(123), notePathID)
						return mockSentMessages, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						require.Contains(t, []int64{1, 2}, chatID)

						// Check if it's an EditMessageTextConfig
						editMsg, ok := msg.(tgbotapi.EditMessageTextConfig)
						require.True(t, ok, "expected EditMessageTextConfig")
						require.Equal(t, "HTML", editMsg.ParseMode)
						require.Contains(t, editMsg.Text, "Updated test note content")
						require.Contains(t, []int64{-1001234567890, -1001234567891}, editMsg.ChatID)
						require.Contains(t, []int{101, 102}, editMsg.MessageID)

						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "success - update message with photo (edit caption)",
			noteID: 123,
			setupEnv: func() *EnvMock {
				// Create note view with image asset
				noteWithImage := &model.NoteView{
					Path:      "/test-note",
					Title:     "Test Note",
					PathID:    123,
					VersionID: 456,
					Content:   content,
					Assets:    map[string]struct{}{"image.jpg": {}},
					AssetReplaces: map[string]*model.NoteAssetReplace{
						"image.jpg": {
							URL: "https://example.com/image.jpg",
						},
					},
					InLinks: map[string]struct{}{},
				}
				noteWithImage.SetAst(doc)

				noteViewsWithImage := &model.NoteViews{
					Map:       map[string]*model.NoteView{"/test-note": noteWithImage},
					List:      []*model.NoteView{noteWithImage},
					Subgraphs: map[string]*model.NoteSubgraph{},
					Version:   "latest",
				}

				return &EnvMock{
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Updated test note content",
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViewsWithImage
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return []db.ListTelegramPublishSentMessagesByNotePathIDRow{mockSentMessages[0]}, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						// Check if it's an EditMessageCaptionConfig
						editMsg, ok := msg.(tgbotapi.EditMessageCaptionConfig)
						require.True(t, ok, "expected EditMessageCaptionConfig for photo message")
						require.Equal(t, "HTML", editMsg.ParseMode)
						require.Contains(t, editMsg.Caption, "Updated test note content")

						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "error - note view not found",
			noteID: 999, // Non-existent note ID
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
				}
			},
			wantErr: true,
			errMsg:  "note view not found for path ID 999",
		},
		{
			name:   "error - no sent messages found",
			noteID: 123,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return []db.ListTelegramPublishSentMessagesByNotePathIDRow{}, nil
					},
				}
			},
			wantErr: true,
			errMsg:  "no sent messages found for note path ID 123",
		},
		{
			name:   "error - failed to get sent messages",
			noteID: 123,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return nil, errors.New("database error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to get sent messages for note: database error",
		},
		{
			name:   "error - failed to convert note to telegram post",
			noteID: 123,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return []db.ListTelegramPublishSentMessagesByNotePathIDRow{mockSentMessages[0]}, nil
					},
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return nil, errors.New("conversion error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to convert note to telegram post: conversion error",
		},
		{
			name:   "error - conversion produced warnings",
			noteID: 123,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return []db.ListTelegramPublishSentMessagesByNotePathIDRow{mockSentMessages[0]}, nil
					},
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Updated test note content",
							Warnings: []string{"unsupported markdown feature"},
						}, nil
					},
				}
			},
			wantErr: true,
			errMsg:  "conversion produced warnings: [unsupported markdown feature]",
		},
		{
			name:   "error - failed to edit telegram message",
			noteID: 123,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return []db.ListTelegramPublishSentMessagesByNotePathIDRow{mockSentMessages[0]}, nil
					},
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Updated test note content",
							Warnings: []string{},
						}, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						return errors.New("telegram API error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to edit telegram message in chat 1: telegram API error",
		},
		{
			name:   "success - skip update when content hash matches",
			noteID: 123,
			setupEnv: func() *EnvMock {
				// Hash for "Updated test note content"
				expectedHash := "79c1b725091cf8266eff024a296e4fd7ff8ce3001aa1958fe591773246f072ad"

				// Create a message with matching hash
				msgWithMatchingHash := db.ListTelegramPublishSentMessagesByNotePathIDRow{
					ChatID:      1,
					MessageID:   101,
					TelegramID:  -1001234567890,
					ContentHash: expectedHash,
				}

				return &EnvMock{
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
						return []db.ListTelegramPublishSentMessagesByNotePathIDRow{msgWithMatchingHash}, nil
					},
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Updated test note content",
							Warnings: []string{},
						}, nil
					},
					SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
						require.Fail(t, "SendTelegramRequest should not be called when content hash matches")
						return nil
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			ctx := context.Background()

			err := updatetelegrampublishpost.Resolve(ctx, env, tt.noteID)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
