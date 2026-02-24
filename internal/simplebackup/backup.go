package simplebackup

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"trip2g/internal/miniostorage"
	"trip2g/internal/model"
)

const (
	retentionCount = 3
	backupPrefix   = "db-backup-"
)

// PerformBackup executes: VACUUM INTO -> gzip -> Upload -> Retention Cleanup.
func (m *Manager) PerformBackup(ctx context.Context) error {
	if !m.mu.TryLock() {
		return errors.New("backup already in progress")
	}
	defer m.mu.Unlock()

	log := m.env.Logger()
	startTime := time.Now()
	log.Info("starting simple backup")

	// 1. VACUUM INTO (Create snapshot)
	tempBackupPath := m.databasePath + fmt.Sprintf(".backup-%d.tmp", startTime.Unix())
	defer os.Remove(tempBackupPath) // Ensure cleanup

	// DB() might be nil during restore phase, but PerformBackup is only called when app is running
	if m.env.DB() == nil {
		return errors.New("database connection is nil")
	}

	_, err := m.env.DB().ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", tempBackupPath))
	if err != nil {
		return fmt.Errorf("VACUUM INTO failed: %w", err)
	}

	// 2. Compress & Upload
	f, err := os.Open(tempBackupPath)
	if err != nil {
		return fmt.Errorf("failed to open temp backup: %w", err)
	}
	defer f.Close()

	pr, pw := io.Pipe()
	go func() {
		gw := gzip.NewWriter(pw)
		_, copyErr := io.Copy(gw, f)
		closeErr := gw.Close()
		if closeErr != nil && copyErr == nil {
			copyErr = closeErr
		}
		pw.CloseWithError(copyErr)
	}()

	objectName := fmt.Sprintf("%s%d.db.gz", backupPrefix, startTime.Unix())

	err = m.env.PutPrivateObject(ctx, pr, objectName)
	if err != nil {
		return fmt.Errorf("failed to upload backup: %w", err)
	}

	// 3. Enforce Retention
	retentionErr := m.enforceRetention(ctx)
	if retentionErr != nil {
		log.Warn("failed to enforce retention policy", "error", retentionErr)
	}

	log.Info("backup completed", "duration", time.Since(startTime))
	return nil
}

func (m *Manager) enforceRetention(ctx context.Context) error {
	objects, err := m.env.ListPrivateObjects(ctx, miniostorage.ListOptions{
		Prefix: backupPrefix,
	})
	if err != nil {
		return err
	}

	backups := filterAndSortBackups(objects)

	if len(backups) > retentionCount {
		toDelete := backups[retentionCount:]
		for _, obj := range toDelete {
			m.env.Logger().Info("deleting old backup", "key", obj.Key)
			deleteErr := m.env.DeletePrivateObject(ctx, obj.Key)
			if deleteErr != nil {
				m.env.Logger().Error("failed to delete old backup", "key", obj.Key, "error", deleteErr)
			}
		}
	}
	return nil
}

// filterAndSortBackups filters objects to valid backup files and sorts newest-first by filename timestamp.
// Unparseable filenames sort to the end (treated as oldest, deleted by retention first).
func filterAndSortBackups(objects []model.PrivateObject) []model.PrivateObject {
	var backups []model.PrivateObject
	for _, obj := range objects {
		if strings.HasPrefix(obj.Key, backupPrefix) && strings.HasSuffix(obj.Key, ".db.gz") {
			backups = append(backups, obj)
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		var ti, tj int64
		ni, _ := fmt.Sscanf(backups[i].Key, backupPrefix+"%d.db.gz", &ti)
		nj, _ := fmt.Sscanf(backups[j].Key, backupPrefix+"%d.db.gz", &tj)
		if ni == 0 || nj == 0 {
			return ni > nj
		}
		return ti > tj
	})

	return backups
}
