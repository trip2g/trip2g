package notion

import (
	"context"
	"fmt"
	"sync"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/notiontypes"
)

type ClientManager struct {
	sync.Mutex

	env     ClientManagerEnv
	config  Config
	clients map[int64]notiontypes.Client
}

type ClientManagerEnv interface {
	NotionIntegration(ctx context.Context, id int64) (db.NotionIntegration, error)
	Logger() logger.Logger
}

func NewClientManager(env ClientManagerEnv, config Config) *ClientManager {
	return &ClientManager{
		env:     env,
		config:  config,
		clients: make(map[int64]notiontypes.Client),
	}
}

func (cm *ClientManager) Get(ctx context.Context, env ClientManagerEnv, id int64) (notiontypes.Client, error) {
	if env == nil {
		env = cm.env
		env.Logger().Debug("using default environment for Notion client manager")
	}

	cm.Lock()
	defer cm.Unlock()

	client, exists := cm.clients[id]
	if !exists {
		integration, err := env.NotionIntegration(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get notion integration: %w", err)
		}

		config := cm.config
		config.Token = integration.SecretToken

		newClient, err := New(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create Notion client: %w", err)
		}

		cm.clients[id] = newClient

		return newClient, nil
	}

	return client, nil
}
