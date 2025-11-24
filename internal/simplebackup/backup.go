package simplebackup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"trip2g/internal/miniostorage"
)

const (
	retentionCount = 3
	backupPrefix   = "db-backup-"
)

// PerformBackup executes: VACUUM INTO -> gzip -> Upload -> Retention Cleanup
func (m *Manager) PerformBackup(ctx context.Context) error {
	if !m.mu.TryLock() {
		return fmt.Errorf("backup already in progress")
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
		return fmt.Errorf("database connection is nil")
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
		gw.Close()
		pw.CloseWithError(copyErr)
	}()

	objectName := fmt.Sprintf("%s%d.db.gz", backupPrefix, startTime.Unix())

	err = m.env.PutPrivateObject(ctx, pr, objectName)
	if err != nil {
		return fmt.Errorf("failed to upload backup: %w", err)
	}

	// 3. Enforce Retention
	if err := m.enforceRetention(ctx); err != nil {
		log.Warn("failed to enforce retention policy", "error", err)
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

	// Filter and sort backups by LastModified (newest first)
	var backups []string
	for _, obj := range objects {
		if strings.HasPrefix(obj.Key, backupPrefix) && strings.HasSuffix(obj.Key, ".db.gz") {
			backups = append(backups, obj.Key)
		}
	}

	// Sort by extracting timestamp from filename
	sort.Slice(backups, func(i, j int) bool {
		// Extract timestamps from filenames for comparison
		// Format: db-backup-{timestamp}.db.gz
		var ti, tj int64
		fmt.Sscanf(filepath.Base(backups[i]), backupPrefix+"%d.db.gz", &ti)
		fmt.Sscanf(filepath.Base(backups[j]), backupPrefix+"%d.db.gz", &tj)
		return ti > tj // Descending (newest first)
	})

	// Delete old backups
	if len(backups) > retentionCount {
		toDelete := backups[retentionCount:]
		for _, key := range toDelete {
			m.env.Logger().Info("deleting old backup", "key", key)
			if err := m.env.DeletePrivateObject(ctx, key); err != nil {
				// Log but continue
				m.env.Logger().Error("failed to delete old backup", "key", key, "error", err)
			}
		}
	}
	return nil
}
