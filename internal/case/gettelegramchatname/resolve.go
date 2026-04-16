package gettelegramchatname

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
)

const (
	positiveCacheTTL      = 7 * 24 * time.Hour
	negativeCacheTTL      = 24 * time.Hour
	refreshRequestBackoff = time.Hour
	defaultRefreshBatch   = 100
)

type LookupResult struct {
	Username string
	Title    string
}

type Env interface {
	Logger() logger.Logger
	Now() time.Time

	GetTelegramChatUsernameByChatID(ctx context.Context, chatID int64) (db.TelegramChatUsername, error)
	ListStaleTelegramChatUsernames(ctx context.Context, arg db.ListStaleTelegramChatUsernamesParams) ([]db.TelegramChatUsername, error)

	UpsertTelegramChatUsername(ctx context.Context, arg db.UpsertTelegramChatUsernameParams) error
	MarkTelegramChatUsernameRefreshRequested(ctx context.Context, arg db.MarkTelegramChatUsernameRefreshRequestedParams) error
	MarkTelegramChatUsernameRefreshError(ctx context.Context, arg db.MarkTelegramChatUsernameRefreshErrorParams) error

	ResolveTelegramChatUsernameViaBot(ctx context.Context, chatID int64) (LookupResult, bool, error)
	ResolveTelegramChatUsernameViaAccount(ctx context.Context, chatID int64) (LookupResult, bool, error)
}

func Resolve(ctx context.Context, env Env, telegramChatID int64) (string, error) {
	now := env.Now()

	row, err := env.GetTelegramChatUsernameByChatID(ctx, telegramChatID)
	switch {
	case err == nil:
		if isFresh(row, now) {
			return row.Username, nil
		}

		maybeRequestRefresh(ctx, env, row, now)
		return row.Username, nil
	case errors.Is(err, sql.ErrNoRows):
		result, found, refreshErr := refreshChatUsername(ctx, env, telegramChatID, now)
		if refreshErr != nil {
			return "", refreshErr
		}
		if !found {
			return "", nil
		}
		return result.Username, nil
	default:
		return "", fmt.Errorf("failed to load telegram chat username cache: %w", err)
	}
}

func RefreshStale(ctx context.Context, env Env, limit int) (int, error) {
	if limit <= 0 {
		limit = defaultRefreshBatch
	}

	now := env.Now()
	rows, err := env.ListStaleTelegramChatUsernames(ctx, db.ListStaleTelegramChatUsernamesParams{
		PositiveStaleBefore: now.Add(-positiveCacheTTL),
		NegativeStaleBefore: now.Add(-negativeCacheTTL),
		Limit:               int64(limit),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list stale telegram chat usernames: %w", err)
	}

	log := logger.WithPrefix(env.Logger(), "gettelegramchatname:")
	refreshed := 0

	for _, row := range rows {
		_, _, refreshErr := refreshChatUsername(ctx, env, row.TelegramChatID, now)
		if refreshErr != nil {
			errMsg := refreshErr.Error()
			log.Warn("failed to refresh telegram chat username",
				"telegram_chat_id", row.TelegramChatID,
				"error", refreshErr,
			)
			_ = env.MarkTelegramChatUsernameRefreshError(ctx, db.MarkTelegramChatUsernameRefreshErrorParams{
				LastError:      &errMsg,
				TelegramChatID: row.TelegramChatID,
			})
			continue
		}

		refreshed++
	}

	return refreshed, nil
}

func refreshChatUsername(ctx context.Context, env Env, telegramChatID int64, now time.Time) (LookupResult, bool, error) {
	result, found, err := resolveViaSources(ctx, env, telegramChatID)
	if err != nil {
		return LookupResult{}, false, err
	}

	if upsertErr := env.UpsertTelegramChatUsername(ctx, db.UpsertTelegramChatUsernameParams{
		TelegramChatID: telegramChatID,
		Username:       result.Username,
		Title:          result.Title,
		RefreshedAt:    now,
	}); upsertErr != nil {
		return LookupResult{}, false, fmt.Errorf("failed to upsert telegram chat username cache: %w", upsertErr)
	}

	return result, found, nil
}

func resolveViaSources(ctx context.Context, env Env, telegramChatID int64) (LookupResult, bool, error) {
	result, found, err := env.ResolveTelegramChatUsernameViaBot(ctx, telegramChatID)
	if err == nil && found {
		return result, true, nil
	}

	accountResult, accountFound, accountErr := env.ResolveTelegramChatUsernameViaAccount(ctx, telegramChatID)
	if accountErr == nil && accountFound {
		return accountResult, true, nil
	}

	switch {
	case err == nil && accountErr == nil:
		if accountResult.Title == "" {
			accountResult.Title = result.Title
		}
		return accountResult, false, nil
	case err != nil:
		return LookupResult{}, false, err
	default:
		return LookupResult{}, false, accountErr
	}
}

func isFresh(row db.TelegramChatUsername, now time.Time) bool {
	ttl := positiveCacheTTL
	if row.Username == "" {
		ttl = negativeCacheTTL
	}

	return row.RefreshedAt.Add(ttl).After(now)
}

func maybeRequestRefresh(ctx context.Context, env Env, row db.TelegramChatUsername, now time.Time) {
	if row.RefreshRequestedAt != nil && row.RefreshRequestedAt.Add(refreshRequestBackoff).After(now) {
		return
	}

	_ = env.MarkTelegramChatUsernameRefreshRequested(ctx, db.MarkTelegramChatUsernameRefreshRequestedParams{
		RefreshRequestedAt: &now,
		TelegramChatID:     row.TelegramChatID,
	})
}
