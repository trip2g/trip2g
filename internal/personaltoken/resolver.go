package personaltoken

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

var ErrInvalidToken = errors.New("invalid or expired personal token")
var ErrBanned = errors.New("user is banned")

type Env interface {
	UserTokenByHash(ctx context.Context, hash string) (db.UserToken, error)
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
	UserBanByUserID(ctx context.Context, userID int64) (*db.UserBan, error)
	UpdateUserTokenLastUsedAt(ctx context.Context, id string) error
}

type cacheEntry struct {
	data      *usertoken.Data
	tokenID   string
	fetchedAt time.Time
}

const cacheTTL = 30 * time.Second
const throttleTTL = 5 * time.Minute

type Resolver struct {
	env   Env
	cache sync.Map // key: token hash (string), value: cacheEntry
	used  sync.Map // key: token ID (string), value: time.Time
}

func NewResolver(env Env) *Resolver {
	return &Resolver{env: env}
}

func (r *Resolver) Resolve(ctx context.Context, plaintext string) (*usertoken.Data, error) {
	data, tokenID, err := r.resolve(ctx, Hash(plaintext))
	if err != nil {
		return nil, err
	}

	// Bans are checked on every request, cache hit or not; the lookup is in memory.
	ban, err := r.env.UserBanByUserID(ctx, int64(data.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to get user ban: %w", err)
	}
	if ban != nil {
		return nil, fmt.Errorf("%w: %s", ErrBanned, ban.Reason)
	}

	r.maybeUpdateLastUsed(tokenID)
	return data, nil
}

func (r *Resolver) resolve(ctx context.Context, hash string) (*usertoken.Data, string, error) {
	if v, ok := r.cache.Load(hash); ok {
		entry := v.(cacheEntry) //nolint:errcheck // type assertion is safe: sync.Map stores only cacheEntry values
		fresh := time.Since(entry.fetchedAt) < cacheTTL
		// A cached admin role is re-checked so a removed admin loses it at once.
		if fresh && entry.data.IsAdmin() {
			_, adminErr := r.env.AdminByUserID(ctx, int64(entry.data.ID))
			fresh = adminErr == nil
		}
		if fresh {
			return entry.data, entry.tokenID, nil
		}
	}

	tok, err := r.env.UserTokenByHash(ctx, hash)
	if err != nil {
		return nil, "", ErrInvalidToken
	}

	role := "user"
	if _, err = r.env.AdminByUserID(ctx, tok.UserID); err == nil {
		role = "admin"
	}

	data := &usertoken.Data{
		ID:   int(tok.UserID),
		Role: role,
	}

	r.cache.Store(hash, cacheEntry{data: data, tokenID: tok.ID, fetchedAt: time.Now()})
	return data, tok.ID, nil
}

// Forget drops every cached resolution of the token with this ID, so the next
// request carrying it is checked against the database again. Call it after a
// token is revoked; without it a revoked token keeps resolving until its cache
// entry expires.
func (r *Resolver) Forget(tokenID string) {
	r.cache.Range(func(key, value any) bool {
		if value.(cacheEntry).tokenID == tokenID { //nolint:errcheck // type assertion is safe: sync.Map stores only cacheEntry values
			r.cache.Delete(key)
		}
		return true
	})
}

func (r *Resolver) maybeUpdateLastUsed(tokenID string) {
	now := time.Now()
	if v, ok := r.used.Load(tokenID); ok {
		if now.Sub(v.(time.Time)) < throttleTTL { //nolint:errcheck // type assertion is safe: sync.Map stores only time.Time values
			return
		}
	}
	r.used.Store(tokenID, now)
	go func() {
		_ = r.env.UpdateUserTokenLastUsedAt(context.Background(), tokenID)
	}()
}
