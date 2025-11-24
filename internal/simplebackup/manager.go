package simplebackup

import (
	"context"
	"database/sql"
	"io"
	"sync"

	"trip2g/internal/logger"
	"trip2g/internal/miniostorage"
	"trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	DB() *sql.DB

	// Storage methods
	ListPrivateObjects(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error)
	DeletePrivateObject(ctx context.Context, objectID string) error
	PutPrivateObject(ctx context.Context, reader io.Reader, objectID string) error
	GetPrivateObject(ctx context.Context, objectID string) (io.ReadCloser, error)
}

type Manager struct {
	mu           sync.Mutex
	env          Env
	databasePath string
}

func New(env Env, databasePath string) *Manager {
	return &Manager{
		env:          env,
		databasePath: databasePath,
	}
}
