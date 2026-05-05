package personaltoken

import (
	"context"
	"errors"
	"sync"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

var ErrInvalidToken = errors.New("invalid or expired personal token")

type Env interface {
	UserTokenByHash(ctx context.Context, hash string) (db.UserToken, error)
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
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
	hash := Hash(plaintext)

	if v, ok := r.cache.Load(hash); ok {
		entry := v.(cacheEntry) //nolint:errcheck
		if time.Since(entry.fetchedAt) < cacheTTL {
			r.maybeUpdateLastUsed(entry.tokenID)
			return entry.data, nil
		}
	}

	tok, err := r.env.UserTokenByHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidToken
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
	r.maybeUpdateLastUsed(tok.ID)
	return data, nil
}

func (r *Resolver) maybeUpdateLastUsed(tokenID string) {
	now := time.Now()
	if v, ok := r.used.Load(tokenID); ok {
		if now.Sub(v.(time.Time)) < throttleTTL { //nolint:errcheck
			return
		}
	}
	r.used.Store(tokenID, now)
	go func() {
		_ = r.env.UpdateUserTokenLastUsedAt(context.Background(), tokenID)
	}()
}
