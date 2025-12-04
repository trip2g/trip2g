package sendtelegramaccountpublishpost_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/sendtelegramaccountpublishpost"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestResolve(t *testing.T) {
	content := []byte("Test note content")
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)

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
	testNote.SetAst(doc)

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-note": testNote,
		},
		List:      []*model.NoteView{testNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "latest",
	}

	mockAccountChats := []db.ListTelegramAccountChatsByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			SessionData:    []byte("session1"),
		},
		{
			AccountID:      2,
			TelegramChatID: -1001234567891,
			SessionData:    []byte("session2"),
		},
	}

	mockInstantChats := []db.ListTelegramAccountInstantChatsByNotePathIDRow{
		{
			AccountID:      1,
			TelegramChatID: -1001234567890,
			SessionData:    []byte("session1"),
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
			name:    "success - regular post to multiple account chats",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Media:    []string{},
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramAccountChatsByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountChatsByNotePathIDRow, error) {
						require.Equal(t, int64(123), notePathID)
						return mockAccountChats, nil
					},
					EnqueueSendTelegramAccountMessageFunc: func(ctx context.Context, params model.TelegramAccountSendPostParams) error {
						require.Equal(t, int64(123), params.NotePathID)
						require.Contains(t, []int64{1, 2}, params.AccountID)
						require.False(t, params.Instant)
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
			name:    "success - instant post to account chats",
			noteID:  123,
			instant: true,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Media:    []string{},
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramAccountInstantChatsByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountInstantChatsByNotePathIDRow, error) {
						require.Equal(t, int64(123), notePathID)
						return mockInstantChats, nil
					},
					EnqueueSendTelegramAccountMessageFunc: func(ctx context.Context, params model.TelegramAccountSendPostParams) error {
						require.Equal(t, int64(123), params.NotePathID)
						require.True(t, params.Instant)
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:    "error - no account chats configured for non-instant",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramAccountChatsByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountChatsByNotePathIDRow, error) {
						return []db.ListTelegramAccountChatsByNotePathIDRow{}, nil
					},
				}
			},
			wantErr: true,
			errMsg:  "no account chats found for note path ID 123",
		},
		{
			name:    "success - no account chats configured for instant (should not error)",
			noteID:  123,
			instant: true,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramAccountInstantChatsByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountInstantChatsByNotePathIDRow, error) {
						return []db.ListTelegramAccountInstantChatsByNotePathIDRow{}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:    "error - note view not found",
			noteID:  999,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
				}
			},
			wantErr: true,
			errMsg:  "note view not found for path ID 999",
		},
		{
			name:    "error - failed to get account chats",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramAccountChatsByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountChatsByNotePathIDRow, error) {
						return nil, errors.New("database error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to get account chats for note: database error",
		},
		{
			name:    "error - failed to convert note to telegram post",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramAccountChatsByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountChatsByNotePathIDRow, error) {
						return mockAccountChats[:1], nil
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
			name:    "error - failed to enqueue account message",
			noteID:  123,
			instant: false,
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
					ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
						return &model.TelegramPost{
							Content:  "Test note content",
							Media:    []string{},
							Warnings: []string{},
						}, nil
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListTelegramAccountChatsByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountChatsByNotePathIDRow, error) {
						return mockAccountChats[:1], nil
					},
					EnqueueSendTelegramAccountMessageFunc: func(ctx context.Context, params model.TelegramAccountSendPostParams) error {
						return errors.New("queue error")
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to enqueue telegram account post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			ctx := context.Background()

			params := model.SendTelegramPublishPostParams{
				NotePathID:        tt.noteID,
				Instant:           tt.instant,
				UpdateLinkedPosts: !tt.instant,
			}
			err := sendtelegramaccountpublishpost.Resolve(ctx, env, params)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
