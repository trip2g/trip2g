package userbans

import (
	"context"
	"fmt"
	"sync"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
)

type Env interface {
	ListAllUserBans(ctx context.Context) ([]db.UserBan, error)
}

type UserBans struct {
	sync.Mutex
	env    Env
	banMap map[int64]db.UserBan
	bans   []db.UserBan
}

// New creates a new UserBans instance.
func New(env Env) *UserBans {
	return &UserBans{
		env: env,
	}
}

func (a *UserBans) UserBanByUserID(ctx context.Context, userID int64) (*db.UserBan, error) {
	a.Lock()
	defer a.Unlock()

	if a.banMap == nil {
		env := a.env

		req, err := appreq.FromCtx(ctx)
		if err != nil && err != appreq.ErrNotFound {
			return nil, fmt.Errorf("failed to get appreq from context: %w", err)
		}

		ctxEnv, ok := req.Env.(Env)
		if ok {
			env = ctxEnv
		}

		userBans, err := env.ListAllUserBans(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get user bans from the db: %w", err)
		}

		a.banMap = make(map[int64]db.UserBan, len(userBans))
		a.bans = userBans

		for _, v := range userBans {
			a.banMap[v.UserID] = v
		}
	}

	ban, ok := a.banMap[userID]
	if !ok {
		return nil, nil
	}

	return &ban, nil
}

func (a *UserBans) ResetBanCache(ctx context.Context) error {
	a.Lock()
	a.banMap = nil
	a.bans = nil
	a.Unlock()

	_, err := a.UserBanByUserID(ctx, 0)

	return err
}
