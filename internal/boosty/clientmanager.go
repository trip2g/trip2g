package boosty

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type ClientManager struct {
	sync.Mutex

	env     ClientManagerEnv
	clients map[int64]*ClientImpl
}

type ClientManagerEnv interface {
	BoostyCredentials(ctx context.Context, id int64) (db.BoostyCredential, error)
	Logger() logger.Logger
}

func NewClientManager(env ClientManagerEnv) *ClientManager {
	return &ClientManager{
		env:     env,
		clients: make(map[int64]*ClientImpl),
	}
}

func (cm *ClientManager) Reset(ctx context.Context, credentialsID int64) {
	cm.env.Logger().Debug("resetting Boosty client", "credentialsID", credentialsID)

	cm.Lock()
	defer cm.Unlock()

	_, exists := cm.clients[credentialsID]
	if exists {
		delete(cm.clients, credentialsID)
	}
}

func (cm *ClientManager) Get(ctx context.Context, env ClientManagerEnv, credentialsID int64) (Client, error) {
	if env == nil {
		env = cm.env
		env.Logger().Debug("using default environment for Boosty client manager")
	}

	cm.Lock()
	defer cm.Unlock()

	client, exists := cm.clients[credentialsID]
	if !exists {
		creds, err := env.BoostyCredentials(ctx, credentialsID)
		if err != nil {
			return nil, fmt.Errorf("failed to get Boosty credentials: %w", err)
		}

		authData := AuthData{}

		err = json.Unmarshal([]byte(creds.AuthData), &authData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Boosty auth data: %w", err)
		}

		authData.DeviceID = creds.DeviceID
		authData.BlogName = creds.BlogName

		newClient, err := NewClient(authData)
		if err != nil {
			return nil, fmt.Errorf("failed to create Patreon client: %w", err)
		}

		cm.clients[credentialsID] = newClient

		return newClient, nil
	}

	return client, nil
}
