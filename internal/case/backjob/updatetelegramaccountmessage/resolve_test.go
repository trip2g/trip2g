package updatetelegramaccountmessage_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"trip2g/internal/case/backjob/updatetelegramaccountmessage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// errRateLimit carries the "Too Many Requests" marker telegram.HandleRateLimit
// looks for; "retry after 0" keeps the retry delay at the 1s floor for tests.
var errRateLimit = errors.New("Too Many Requests: retry after 0") //nolint:staticcheck // mirrors literal Telegram API error text

func updateAccountRetryParams() model.TelegramAccountUpdatePostParams {
	return model.TelegramAccountUpdatePostParams{
		TelegramAccountSendPostParams: model.TelegramAccountSendPostParams{
			NotePathID:     123,
			AccountID:      1,
			TelegramChatID: 789,
			Post: model.TelegramPost{
				Content: "Updated message",
				Media:   []string{},
			},
		},
		MessageID: 111,
	}
}

func expectedTextContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// updateAccountRetryEnv builds an EnvMock where GetTelegramAccountByID is the
// rate-limit failure point; the ContentHash counter lets a later attempt skip
// past editing entirely (hash already matches) to reach a real success
// without ever touching the real tgtd network client.
func updateAccountRetryEnv(getAccountFn func(ctx context.Context, id int64) (db.TelegramAccount, error)) *EnvMock {
	return &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		GetTelegramPublishSentAccountMessagePostTypeFunc: func(ctx context.Context, arg db.GetTelegramPublishSentAccountMessagePostTypeParams) (string, error) {
			return "text", nil
		},
		GetTelegramAccountByIDFunc: getAccountFn,
	}
}

func TestResolve_RateLimit_RetriedThenSucceeds(t *testing.T) {
	hashCalls := 0
	getAccountCalls := 0
	newHash := expectedTextContentHash("Updated message")

	env := updateAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		getAccountCalls++
		return db.TelegramAccount{}, errRateLimit
	})
	env.GetTelegramPublishSentAccountMessageContentHashFunc = func(ctx context.Context, arg db.GetTelegramPublishSentAccountMessageContentHashParams) (string, error) {
		hashCalls++
		if hashCalls == 1 {
			return "old_hash", nil // differs — proceed to edit, which will rate-limit
		}
		return newHash, nil // matches by the time we retry — skip cleanly
	}

	err := updatetelegramaccountmessage.Resolve(context.Background(), env, updateAccountRetryParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hashCalls != 2 {
		t.Errorf("expected 2 resolve attempts (1 retry), got %d", hashCalls)
	}
	if getAccountCalls != 1 {
		t.Errorf("expected GetTelegramAccountByID called once (rate-limited then skipped), got %d", getAccountCalls)
	}
}

func TestResolve_RateLimit_RetriesExhausted(t *testing.T) {
	calls := 0
	env := updateAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		calls++
		return db.TelegramAccount{}, errRateLimit
	})
	env.GetTelegramPublishSentAccountMessageContentHashFunc = func(ctx context.Context, arg db.GetTelegramPublishSentAccountMessageContentHashParams) (string, error) {
		return "old_hash", nil
	}

	err := updatetelegramaccountmessage.Resolve(context.Background(), env, updateAccountRetryParams())
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
	env := updateAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		calls++
		cancel() // cancel before the retry wait
		return db.TelegramAccount{}, errRateLimit
	})
	env.GetTelegramPublishSentAccountMessageContentHashFunc = func(ctx context.Context, arg db.GetTelegramPublishSentAccountMessageContentHashParams) (string, error) {
		return "old_hash", nil
	}

	err := updatetelegramaccountmessage.Resolve(ctx, env, updateAccountRetryParams())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 attempt before ctx cancel, got %d", calls)
	}
}

func TestResolve_NonRateLimit_NoRetry(t *testing.T) {
	calls := 0
	env := updateAccountRetryEnv(func(ctx context.Context, id int64) (db.TelegramAccount, error) {
		calls++
		return db.TelegramAccount{}, errors.New("some other telegram error")
	})
	env.GetTelegramPublishSentAccountMessageContentHashFunc = func(ctx context.Context, arg db.GetTelegramPublishSentAccountMessageContentHashParams) (string, error) {
		return "old_hash", nil
	}

	err := updatetelegramaccountmessage.Resolve(context.Background(), env, updateAccountRetryParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", calls)
	}
}
