package patreon

import (
	"fmt"
	"sync"
)

type ClientManager struct {
	sync.Mutex

	clients map[string]*Client
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*Client),
	}
}

func (cm *ClientManager) Get(creatorAccessToken string) (*Client, error) {
	cm.Lock()
	defer cm.Unlock()

	client, exists := cm.clients[creatorAccessToken]
	if !exists {
		cfg := ClientConfig{
			CreatorAccessToken: creatorAccessToken,
		}

		newClient, err := NewClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create Patreon client: %w", err)
		}

		cm.clients[creatorAccessToken] = newClient

		return newClient, nil
	}

	return client, nil
}
