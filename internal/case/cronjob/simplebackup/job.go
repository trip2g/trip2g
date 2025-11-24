package simplebackup

import (
	"context"
	"trip2g/internal/simplebackup"
)

type Job struct{}

func (j *Job) Name() string {
	return "simple_backup"
}

func (j *Job) Schedule() string {
	return "0 0 * * * *" // Every hour at :00
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

// Env interface that allows accessing the backup manager
type Env interface {
	BackupManager() *simplebackup.Manager
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	e, ok := env.(Env)
	if !ok {
		return nil, nil // Config not enabled or invalid env
	}

	mgr := e.BackupManager()
	if mgr == nil {
		return nil, nil // Backup disabled
	}

	return nil, mgr.PerformBackup(ctx)
}
