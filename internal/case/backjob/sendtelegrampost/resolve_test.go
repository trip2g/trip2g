package sendtelegrampost_test

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/case/backjob/sendtelegrampost"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrampost_test . Env

type Env interface {
	SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error)
	InsertTelegramPublishSentMessage(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error
}

func TestResolve_Success_TextOnly(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Test message",
			Images:  []string{},
		},
		Instant: false,
	}

	env := &EnvMock{
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			if chatID != 456 {
				t.Errorf("expected chatID 456, got %d", chatID)
			}
			return 111, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			if arg.NotePathID != 123 {
				t.Errorf("expected NotePathID 123, got %d", arg.NotePathID)
			}
			if arg.ChatID != 456 {
				t.Errorf("expected ChatID 456, got %d", arg.ChatID)
			}
			if arg.MessageID != 111 {
				t.Errorf("expected MessageID 111, got %d", arg.MessageID)
			}
			if arg.Instant {
				t.Error("expected Instant false, got true")
			}
			if arg.Content != "Test message" {
				t.Errorf("expected Content 'Test message', got %s", arg.Content)
			}
			if arg.ContentHash == "" {
				t.Error("expected ContentHash not empty")
			}
			return nil
		},
	}

	err := sendtelegrampost.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env.SendTelegramMessageCalls()) != 1 {
		t.Errorf("expected SendTelegramMessage to be called once, got %d", len(env.SendTelegramMessageCalls()))
	}

	if len(env.InsertTelegramPublishSentMessageCalls()) != 1 {
		t.Errorf("expected InsertTelegramPublishSentMessage to be called once, got %d", len(env.InsertTelegramPublishSentMessageCalls()))
	}
}

func TestResolve_Success_Instant(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Instant message",
			Images:  []string{},
		},
		Instant: true,
	}

	env := &EnvMock{
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 222, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			if !arg.Instant {
				t.Error("expected Instant true, got false")
			}
			return nil
		},
	}

	err := sendtelegrampost.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolve_Success_WithImages(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Message with image",
			Images:  []string{"https://example.com/image.jpg"},
		},
		Instant: false,
	}

	env := &EnvMock{
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			// Should send photo
			return 333, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			if arg.MessageID != 333 {
				t.Errorf("expected MessageID 333, got %d", arg.MessageID)
			}
			return nil
		},
	}

	err := sendtelegrampost.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolve_Error_SendMessage(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Test message",
			Images:  []string{},
		},
		Instant: false,
	}

	expectedErr := errors.New("telegram API error")

	env := &EnvMock{
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 0, expectedErr
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			t.Error("should not insert sent message when send fails")
			return nil
		},
	}

	err := sendtelegrampost.Resolve(ctx, env, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got %v", expectedErr, err)
	}

	if len(env.InsertTelegramPublishSentMessageCalls()) != 0 {
		t.Errorf("expected InsertTelegramPublishSentMessage not to be called, got %d calls", len(env.InsertTelegramPublishSentMessageCalls()))
	}
}

func TestResolve_Error_InsertSentMessage(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Test message",
			Images:  []string{},
		},
		Instant: false,
	}

	expectedErr := errors.New("database error")

	env := &EnvMock{
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 444, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			return expectedErr
		},
	}

	err := sendtelegrampost.Resolve(ctx, env, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got %v", expectedErr, err)
	}
}

func TestResolve_ContentHash_Consistency(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Consistent content",
			Images:  []string{},
		},
		Instant: false,
	}

	var firstHash string

	env := &EnvMock{
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 555, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			if firstHash == "" {
				firstHash = arg.ContentHash
			}
			return nil
		},
	}

	// Run twice with same content
	err := sendtelegrampost.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	secondHash := ""
	env.InsertTelegramPublishSentMessageFunc = func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
		secondHash = arg.ContentHash
		return nil
	}

	err = sendtelegrampost.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if firstHash != secondHash {
		t.Errorf("expected consistent hash, got %s and %s", firstHash, secondHash)
	}
}

func TestResolve_EmptyContent(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "",
			Images:  []string{},
		},
		Instant: false,
	}

	env := &EnvMock{
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 666, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			if arg.Content != "" {
				t.Errorf("expected empty content, got %s", arg.Content)
			}
			if arg.ContentHash == "" {
				t.Error("expected ContentHash not empty even for empty content")
			}
			return nil
		},
	}

	err := sendtelegrampost.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
