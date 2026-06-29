package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"trip2g/internal/case/admin/deletesecret"
	"trip2g/internal/case/admin/getsecret"
	"trip2g/internal/case/admin/setsecret"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/simplebackup"
)

func (a *app) GitCommit() string {
	return GitCommit
}

func (a *app) DatabaseFilePath() string {
	return a.config.DatabaseFile
}

func (a *app) CronJobsAllowEdit() bool {
	return a.config.CronJobs.AllowEdit
}

func (a *app) SiteTitleTemplate() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.SiteConfig(ctx).SiteTitleTemplate
}

// LoadSiteConfig implements noteloader.Env interface.
func (a *app) LoadSiteConfig(ctx context.Context) (model.SiteConfig, error) {
	return a.SiteConfig(ctx), nil
}

func (a *app) CurrentTx() *sql.Tx {
	return a.currentTx
}

// WithTransaction runs the given function within a database transaction.
// fn should return true to commit the transaction, false to rollback.
func (a *app) WithTransaction(ctx context.Context, fn func(context.Context, *app) (bool, error)) error {
	// not sure but I guess transactions should run on writeConn
	tx, err := a.writeConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to BeginTx: %w", err)
	}

	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			a.log.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}()

	queries := db.NewWriteQueries(db.WithLogger(tx, logger.WithPrefix(a.log, "tx")))

	newEnv := *a
	newEnv.queries = queries.Queries
	newEnv.Queries = queries.Queries
	newEnv.WriteQueries = queries
	newEnv.currentTx = tx

	// Store transactional env in context so background jobs can access it
	txCtx := context.WithValue(ctx, txEnvKey, &newEnv)

	commit, err := fn(txCtx, &newEnv)
	if commit {
		commitErr := tx.Commit()
		if commitErr != nil {
			return fmt.Errorf("failed to commit transaction: %w", commitErr)
		}
	} else {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			a.log.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}

	return err
}

func (a *app) Now() time.Time {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timezone := a.SiteConfig(ctx).Timezone

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
		a.log.Error("failed to load timezone location", "timezone", timezone, "error", err)
	}

	return time.Now().In(loc)
}

func (a *app) Logger() logger.Logger {
	return a.log
}

func (a *app) CronScheduleOverride(jobName string) string {
	return a.config.CronScheduleOverride(jobName)
}

func (a *app) LogLevel() string {
	return a.config.LogLevel
}

func (a *app) Features() features.Features {
	return a.config.Features
}

func (a *app) AuditLogger() logger.Logger {
	return a.auditLogger
}

func (a *app) DB() *sql.DB {
	return a.conn
}

func (a *app) PublicURL() string {
	return a.config.PublicURL
}

func (a *app) GetSecretValue(ctx context.Context, key string) (string, error) {
	return getsecret.Resolve(ctx, a, key)
}

func (a *app) GetSecretValues(ctx context.Context, like string) (map[string]string, error) {
	keys, err := a.Queries.ListSecretKeys(ctx, like)
	if err != nil || len(keys) == 0 {
		return nil, err
	}
	result := make(map[string]string, len(keys))
	for _, k := range keys {
		val, decErr := getsecret.Resolve(ctx, a, k)
		if decErr != nil {
			return nil, decErr
		}
		result[k] = val
	}
	return result, nil
}

func (a *app) SetSecretValue(ctx context.Context, key, value string) error {
	return setsecret.Resolve(ctx, a, key, value)
}

func (a *app) DeleteSecretValue(ctx context.Context, key string) error {
	return deletesecret.Resolve(ctx, a, key)
}

func (a *app) IsDevMode() bool {
	return a.config.DevMode
}

func (a *app) MaxRequestBodySize() int {
	return a.config.MaxRequestBodySize
}

func (a *app) MaxActiveSignInCodes() int64 {
	return int64(a.config.MaxActiveSignInCodes)
}

// BackupManager returns the backup manager for cronjob env interface.
func (a *app) BackupManager() *simplebackup.Manager {
	return a.simpleBackup
}
