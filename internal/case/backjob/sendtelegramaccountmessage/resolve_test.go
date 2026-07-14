package sendtelegramaccountmessage_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/backjob/sendtelegramaccountmessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// errRateLimit carries the "Too Many Requests" marker telegram.HandleRateLimit
// looks for; "retry after 0" keeps the retry delay at the 1s floor for tests.
var errRateLimit = errors.New("Too Many Requests: retry after 0") //nolint:staticcheck // mirrors literal Telegram API error text

func sendAccountRetryParams() model.TelegramAccountSendPostParams {
	return model.TelegramAccountSendPostParams{
		NotePathID:     123,
		AccountID:      1,
		TelegramChatID: 789,
		Post: model.TelegramPost{
			Content: "Test message",
			Media:   []string{},
		},
	}
}

// sendAccountRetryEnv builds an EnvMock where GetTelegramAccountByID is the
// rate-limit failure point; the CheckExists counter lets a later attempt skip
// past sending entirely (already-sent) to reach a real success without ever
// touching the real tgtd network client.
func sendAccountRetryEnv(getAccountFn func(ctx context.Context, id int64) (db.TelegramAccount, error)) *EnvMock {
	return &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		CheckTelegramPublishSentAccountMessageExistsFunc: func(ctx context.Context, arg db.CheckTelegramPublishSentAccountMessageExistsParams) (int64, error) {
			return 0, nil
		},
		GetTelegramAccountByIDFunc: getAccountFn,
	}
}

func TestResolve_RateLimit_RetriedThenSucceeds(t *testing.T) {
	checkCalls := 0
	getAccountCalls := 0

	env := sendAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		getAccountCalls++
		return db.TelegramAccount{}, errRateLimit
	})
	env.CheckTelegramPublishSentAccountMessageExistsFunc = func(ctx context.Context, arg db.CheckTelegramPublishSentAccountMessageExistsParams) (int64, error) {
		checkCalls++
		if checkCalls == 1 {
			return 0, nil // not sent yet — proceed to send, which will rate-limit
		}
		return 999, nil // already sent by the time we retry — skip cleanly
	}

	err := sendtelegramaccountmessage.Resolve(context.Background(), env, sendAccountRetryParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if checkCalls != 2 {
		t.Errorf("expected 2 resolve attempts (1 retry), got %d", checkCalls)
	}
	if getAccountCalls != 1 {
		t.Errorf("expected GetTelegramAccountByID called once (rate-limited then skipped), got %d", getAccountCalls)
	}
}

func TestResolve_RateLimit_RetriesExhausted(t *testing.T) {
	calls := 0
	env := sendAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		calls++
		return db.TelegramAccount{}, errRateLimit
	})

	err := sendtelegramaccountmessage.Resolve(context.Background(), env, sendAccountRetryParams())
	if err == nil {
		t.Fatal("expected error after retries exhausted, got nil")
	}
	if calls != 3 {
		t.Errorf("expected 3 attempts (max), got %d", calls)
	}
}

func TestResolve_RateLimit_CtxCancelledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	env := sendAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		calls++
		cancel() // cancel before the retry wait
		return db.TelegramAccount{}, errRateLimit
	})

	err := sendtelegramaccountmessage.Resolve(ctx, env, sendAccountRetryParams())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 attempt before ctx cancel, got %d", calls)
	}
}

func TestResolve_NonRateLimit_NoRetry(t *testing.T) {
	calls := 0
	env := sendAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		calls++
		return db.TelegramAccount{}, errors.New("some other telegram error")
	})

	err := sendtelegramaccountmessage.Resolve(context.Background(), env, sendAccountRetryParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", calls)
	}
}
