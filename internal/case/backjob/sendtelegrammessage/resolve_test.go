package sendtelegrammessage_test

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/case/backjob/sendtelegrammessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg sendtelegrammessage_test . Env

type Env interface {
	SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error)
	InsertTelegramPublishSentMessage(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error
	CheckTelegramPublishSentMessageExists(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error)
	LatestNoteViews() *model.NoteViews
	UpdateTelegramPublishPost(ctx context.Context, notePathID int64) error
	Logger() logger.Logger
}

func TestResolve_Success_TextOnly(t *testing.T) {
	ctx := context.Background()

	params := model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Test message",
			Media:   []string{},
		},
		Instant: false,
	}

	env := &EnvMock{
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil // Message doesn't exist yet
		},
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
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
	}

	err := sendtelegrammessage.Resolve(ctx, env, params)
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
			Media:   []string{},
		},
		Instant: true,
	}

	env := &EnvMock{
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 222, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			if !arg.Instant {
				t.Error("expected Instant true, got false")
			}
			return nil
		},
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
	}

	err := sendtelegrammessage.Resolve(ctx, env, params)
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
			Media:   []string{"https://example.com/image.jpg"},
		},
		Instant: false,
	}

	env := &EnvMock{
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
		TelegramCaptionLengthLimitFunc: func(ctx context.Context, accountID *int64) int {
			return 1024
		},
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
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
	}

	err := sendtelegrammessage.Resolve(ctx, env, params)
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
			Media:   []string{},
		},
		Instant: false,
	}

	expectedErr := errors.New("telegram API error")

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 0, expectedErr
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			t.Error("should not insert sent message when send fails")
			return nil
		},
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
	}

	err := sendtelegrammessage.Resolve(ctx, env, params)
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
			Media:   []string{},
		},
		Instant: false,
	}

	expectedErr := errors.New("database error")

	env := &EnvMock{
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 444, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			return expectedErr
		},
	}

	err := sendtelegrammessage.Resolve(ctx, env, params)
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
			Media:   []string{},
		},
		Instant: false,
	}

	var firstHash string

	env := &EnvMock{
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
		SendTelegramMessageFunc: func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
			return 555, nil
		},
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			if firstHash == "" {
				firstHash = arg.ContentHash
			}
			return nil
		},
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
	}

	// Run twice with same content
	err := sendtelegrammessage.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	secondHash := ""
	env.CheckTelegramPublishSentMessageExistsFunc = func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
		return 0, nil
	}
	env.InsertTelegramPublishSentMessageFunc = func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
		secondHash = arg.ContentHash
		return nil
	}

	err = sendtelegrammessage.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if firstHash != secondHash {
		t.Errorf("expected consistent hash, got %s and %s", firstHash, secondHash)
	}
}

// rateLimitErr carries the "Too Many Requests" marker telegram.HandleRateLimit
// looks for; "retry after 0" keeps the retry delay at the 1s floor for tests.
var rateLimitErr = errors.New("Too Many Requests: retry after 0")

func sendRetryParams() model.TelegramSendPostParams {
	return model.TelegramSendPostParams{
		NotePathID:     123,
		DBChatID:       456,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Test message",
			Media:   []string{},
		},
	}
}

func sendRetryEnv(sendFn func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error)) *EnvMock {
	return &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
		SendTelegramMessageFunc: sendFn,
		InsertTelegramPublishSentMessageFunc: func(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error {
			return nil
		},
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
	}
}

func TestResolve_RateLimit_RetriedThenSucceeds(t *testing.T) {
	calls := 0
	env := sendRetryEnv(func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
		calls++
		if calls == 1 {
			return 0, rateLimitErr
		}
		return 111, nil
	})

	err := sendtelegrammessage.Resolve(context.Background(), env, sendRetryParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 send attempts (1 retry), got %d", calls)
	}
	if len(env.InsertTelegramPublishSentMessageCalls()) != 1 {
		t.Errorf("expected message inserted once after retry, got %d", len(env.InsertTelegramPublishSentMessageCalls()))
	}
}

func TestResolve_RateLimit_RetriesExhausted(t *testing.T) {
	env := sendRetryEnv(func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
		return 0, rateLimitErr
	})

	err := sendtelegrammessage.Resolve(context.Background(), env, sendRetryParams())
	if err == nil {
		t.Fatal("expected error after retries exhausted, got nil")
	}
	if got := len(env.SendTelegramMessageCalls()); got != 3 {
		t.Errorf("expected 3 send attempts (max), got %d", got)
	}
}

func TestResolve_RateLimit_CtxCancelledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	env := sendRetryEnv(func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
		cancel() // cancel before the retry wait
		return 0, rateLimitErr
	})

	err := sendtelegrammessage.Resolve(ctx, env, sendRetryParams())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if got := len(env.SendTelegramMessageCalls()); got != 1 {
		t.Errorf("expected 1 send attempt before ctx cancel, got %d", got)
	}
}

func TestResolve_NonRateLimit_NoRetry(t *testing.T) {
	env := sendRetryEnv(func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
		return 0, errors.New("some other telegram error")
	})

	err := sendtelegrammessage.Resolve(context.Background(), env, sendRetryParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := len(env.SendTelegramMessageCalls()); got != 1 {
		t.Errorf("expected 1 send attempt (no retry), got %d", got)
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
			Media:   []string{},
		},
		Instant: false,
	}

	env := &EnvMock{
		CheckTelegramPublishSentMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error) {
			return 0, nil
		},
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
		ClearTelegramPublishNoteLastErrorFunc: func(ctx context.Context, notePathID int64) error {
			return nil
		},
		SetTelegramPublishNoteLastErrorFunc: func(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error {
			return nil
		},
	}

	err := sendtelegrammessage.Resolve(ctx, env, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
