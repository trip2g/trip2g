package updatetelegrammessage_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/backjob/updatetelegrammessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestResolve_Success_TextMessage(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramUpdatePostParams{
		TelegramSendPostParams: model.TelegramSendPostParams{
			NotePathID:     123,
			DBChatID:       456,
			TelegramChatID: 789,
			Post: model.TelegramPost{
				Content: "Updated message",
				Media:   []string{},
			},
			Instant:           false,
			UpdateLinkedPosts: false,
		},
		MessageID: 111,
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
			return "text", nil
		},
		GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
			// Return different hash to trigger update
			return "old_hash", nil
		},
		SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
			if chatID != 456 {
				t.Errorf("expected chatID 456, got %d", chatID)
			}

			// Verify it's an edit message text request
			editMsg, ok := msg.(tgbotapi.EditMessageTextConfig)
			if !ok {
				t.Errorf("expected EditMessageTextConfig, got %T", msg)
			}

			if editMsg.Text != "Updated message" {
				t.Errorf("expected text 'Updated message', got %s", editMsg.Text)
			}

			if editMsg.ParseMode != "HTML" {
				t.Errorf("expected ParseMode 'HTML', got %s", editMsg.ParseMode)
			}

			if editMsg.ChatID != 789 {
				t.Errorf("expected ChatID 789, got %d", editMsg.ChatID)
			}

			if editMsg.MessageID != 111 {
				t.Errorf("expected MessageID 111, got %d", editMsg.MessageID)
			}

			return nil
		},
		UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
			if arg.NotePathID != 123 {
				t.Errorf("expected NotePathID 123, got %d", arg.NotePathID)
			}
			if arg.ChatID != 456 {
				t.Errorf("expected ChatID 456, got %d", arg.ChatID)
			}
			if arg.MessageID != 111 {
				t.Errorf("expected MessageID 111, got %d", arg.MessageID)
			}
			if arg.Content != "Updated message" {
				t.Errorf("expected Content 'Updated message', got %s", arg.Content)
			}
			if arg.ContentHash == "" {
				t.Error("expected ContentHash not empty")
			}
			return nil
		},
	}

	err := updatetelegrammessage.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env.SendTelegramRequestCalls()) != 1 {
		t.Errorf("expected SendTelegramRequest to be called once, got %d", len(env.SendTelegramRequestCalls()))
	}

	if len(env.UpdateTelegramPublishSentMessageContentCalls()) != 1 {
		t.Errorf("expected UpdateTelegramPublishSentMessageContent to be called once, got %d", len(env.UpdateTelegramPublishSentMessageContentCalls()))
	}
}

func TestResolve_Success_PhotoMessage(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramUpdatePostParams{
		TelegramSendPostParams: model.TelegramSendPostParams{
			NotePathID:     123,
			DBChatID:       456,
			TelegramChatID: 789,
			Post: model.TelegramPost{
				Content: "Updated caption",
				Media:   []string{"https://example.com/image.jpg"},
			},
			Instant:           false,
			UpdateLinkedPosts: false,
		},
		MessageID: 222,
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
			return "photo", nil
		},
		GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
			return "old_hash", nil
		},
		TelegramCaptionLengthLimitFunc: func(ctx context.Context, accountID *int64) int {
			return 1024
		},
		SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
			// Verify it's an edit message caption request
			editMsg, ok := msg.(tgbotapi.EditMessageCaptionConfig)
			if !ok {
				t.Errorf("expected EditMessageCaptionConfig, got %T", msg)
			}

			if editMsg.Caption != "Updated caption" {
				t.Errorf("expected caption 'Updated caption', got %s", editMsg.Caption)
			}

			if editMsg.ParseMode != "HTML" {
				t.Errorf("expected ParseMode 'HTML', got %s", editMsg.ParseMode)
			}

			return nil
		},
		UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
			if arg.Content != "Updated caption" {
				t.Errorf("expected Content 'Updated caption', got %s", arg.Content)
			}
			return nil
		},
	}

	err := updatetelegrammessage.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolve_Error_SendTelegramRequest(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramUpdatePostParams{
		TelegramSendPostParams: model.TelegramSendPostParams{
			NotePathID:     123,
			DBChatID:       456,
			TelegramChatID: 789,
			Post: model.TelegramPost{
				Content: "Updated message",
				Media:   []string{},
			},
		},
		MessageID: 111,
	}

	expectedErr := errors.New("telegram API error")

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
			return "text", nil
		},
		GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
			return "old_hash", nil
		},
		SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
			return expectedErr
		},
		UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
			t.Error("should not update DB when Telegram request fails")
			return nil
		},
	}

	err := updatetelegrammessage.Resolve(ctx, env, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got %v", expectedErr, err)
	}

	if len(env.UpdateTelegramPublishSentMessageContentCalls()) != 0 {
		t.Errorf("expected UpdateTelegramPublishSentMessageContent not to be called, got %d calls", len(env.UpdateTelegramPublishSentMessageContentCalls()))
	}
}

func TestResolve_Success_ContentSameError(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramUpdatePostParams{
		TelegramSendPostParams: model.TelegramSendPostParams{
			NotePathID:     123,
			DBChatID:       456,
			TelegramChatID: 789,
			Post: model.TelegramPost{
				Content: "Same message",
				Media:   []string{},
			},
		},
		MessageID: 111,
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
			return "text", nil
		},
		GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
			return "old_hash", nil
		},
		SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
			// Telegram returns this error when content is the same
			return errors.New(
				"Bad Request: message is not modified: " +
					"specified new message content and reply markup are exactly the same as " +
					"a current content and reply markup of the message",
			)
		},
		UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
			// Should still update the hash in DB even when content is same
			return nil
		},
	}

	err := updatetelegrammessage.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error when content is same: %v", err)
	}

	// Should still update DB
	if len(env.UpdateTelegramPublishSentMessageContentCalls()) != 1 {
		t.Errorf("expected UpdateTelegramPublishSentMessageContent to be called once, got %d", len(env.UpdateTelegramPublishSentMessageContentCalls()))
	}
}

func TestResolve_Error_UpdateDB(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramUpdatePostParams{
		TelegramSendPostParams: model.TelegramSendPostParams{
			NotePathID:     123,
			DBChatID:       456,
			TelegramChatID: 789,
			Post: model.TelegramPost{
				Content: "Updated message",
				Media:   []string{},
			},
		},
		MessageID: 111,
	}

	expectedErr := errors.New("database error")

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
			return "text", nil
		},
		GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
			return "old_hash", nil
		},
		SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
			return nil
		},
		UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
			return expectedErr
		},
	}

	err := updatetelegrammessage.Resolve(ctx, env, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got %v", expectedErr, err)
	}
}

func TestResolve_SkipUpdate_SameContentHash(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramUpdatePostParams{
		TelegramSendPostParams: model.TelegramSendPostParams{
			NotePathID:     123,
			DBChatID:       456,
			TelegramChatID: 789,
			Post: model.TelegramPost{
				Content: "Same message",
				Media:   []string{},
			},
		},
		MessageID: 111,
	}

	// Calculate expected hash for "Same message"
	// This will be the same hash returned by GetTelegramPublishSentMessageContentHashFunc
	expectedHash := "23f5a7f782386a8c730eb36f2327c14f6bea6fa98226ea9a20dcdc8b76f008a4"

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		GetTelegramPublishSentMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error) {
			return "text", nil
		},
		GetTelegramPublishSentMessageContentHashFunc: func(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error) {
			// Return same hash as new content
			return expectedHash, nil
		},
		SendTelegramRequestFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
			t.Error("should not send telegram request when content hash is same")
			return nil
		},
		UpdateTelegramPublishSentMessageContentFunc: func(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error {
			t.Error("should not update DB when content hash is same")
			return nil
		},
	}

	err := updatetelegrammessage.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no Telegram request was made
	if len(env.SendTelegramRequestCalls()) != 0 {
		t.Errorf("expected SendTelegramRequest not to be called, got %d calls", len(env.SendTelegramRequestCalls()))
	}

	// Verify no DB update was made
	if len(env.UpdateTelegramPublishSentMessageContentCalls()) != 0 {
		t.Errorf("expected UpdateTelegramPublishSentMessageContent not to be called, got %d calls", len(env.UpdateTelegramPublishSentMessageContentCalls()))
	}
}
