package patreon

import (
	"context"
	"fmt"
	"sync"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type ClientManager struct {
	sync.Mutex

	env     ClientManagerEnv
	clients map[int64]Client
}

type ClientManagerEnv interface {
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	Logger() logger.Logger
}

func NewClientManager(env ClientManagerEnv) *ClientManager {
	return &ClientManager{
		env:     env,
		clients: make(map[int64]Client),
	}
}

func (cm *ClientManager) Get(ctx context.Context, env ClientManagerEnv, id int64) (Client, error) {
	if env == nil {
		env = cm.env
		env.Logger().Debug("using default environment for Patreon client manager")
	}

	cm.Lock()
	defer cm.Unlock()

	client, exists := cm.clients[id]
	if !exists {
		creds, err := env.PatreonCredentials(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get patreon credentials: %w", err)
		}

		cfg := ClientConfig{
			CreatorAccessToken: creds.CreatorAccessToken,
		}

		newClient, err := NewClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create Patreon client: %w", err)
		}

		cm.clients[id] = newClient

		return newClient, nil
	}

	return client, nil
}
