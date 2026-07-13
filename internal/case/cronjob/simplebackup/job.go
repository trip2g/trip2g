package simplebackup

import (
	"context"
	"trip2g/internal/cronjobs"
	"trip2g/internal/simplebackup"
)

type job struct {
	env Env
}

func New(env Env) cronjobs.Job {
	return &job{env: env}
}

func (j *job) Name() string {
	return "simple_backup"
}

func (j *job) Schedule() string {
	return "0 0 * * * *" // Every hour at :00
}

func (j *job) ExecuteAfterStart() bool {
	return false
}

// Env interface that allows accessing the backup manager. The job delegates to
// internal/simplebackup.Manager; the logic stays in Execute rather than a
// resolve.go — a thin wrapper would add ceremony without testable behavior.
type Env interface {
	BackupManager() *simplebackup.Manager
}

func (j *job) Execute(ctx context.Context) (any, error) {
	mgr := j.env.BackupManager()
	if mgr == nil {
		return nil, nil // Backup disabled
	}

	return nil, mgr.PerformBackup(ctx)
}
