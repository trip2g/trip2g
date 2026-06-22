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
	// genMarker separates the generation counter from the unix timestamp in the
	// object name, e.g. "db-backup-gen7-1700000000.db.gz".
	genMarker = "gen"
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

	// Compute the next generation from the existing S3 backups so the counter is
	// globally monotonic, independent of the (possibly stale) local generation and
	// of wall clocks. The new generation is written into the source DB header
	// (PRAGMA user_version) BEFORE the VACUUM INTO so the snapshot carries it.
	//
	// Caveat: this assumes a single writer at a time (the writer-slot /
	// Deployment replicas=1 model). Two concurrent writers on separate DB files
	// could read the same max generation and both compute the same gen+1; true
	// safety would require an S3 conditional put (compare-and-set on the object
	// name). That is intentionally not implemented here.
	newGen, err := m.nextGeneration(ctx)
	if err != nil {
		return fmt.Errorf("failed to compute backup generation: %w", err)
	}

	_, err = m.env.DB().ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", newGen))
	if err != nil {
		return fmt.Errorf("failed to set user_version: %w", err)
	}

	_, err = m.env.DB().ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", tempBackupPath))
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

	objectName := fmt.Sprintf("%s%s%d-%d.db.gz", backupPrefix, genMarker, newGen, startTime.Unix())

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

// nextGeneration returns (max generation across existing S3 backups) + 1.
// Legacy backups without a generation are treated as generation 0, so the first
// gen'd backup after a legacy history is generation 1.
func (m *Manager) nextGeneration(ctx context.Context) (int, error) {
	objects, err := m.env.ListPrivateObjects(ctx, miniostorage.ListOptions{
		Prefix: backupPrefix,
	})
	if err != nil {
		return 0, err
	}

	maxGen := 0
	for _, obj := range objects {
		if !strings.HasPrefix(obj.Key, backupPrefix) || !strings.HasSuffix(obj.Key, ".db.gz") {
			continue
		}
		if g := backupGeneration(obj.Key); g > maxGen {
			maxGen = g
		}
	}
	return maxGen + 1, nil
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

// filterAndSortBackups filters objects to valid backup files and sorts newest-first.
// Ordering is by generation descending, with the unix timestamp as a tiebreaker.
// Legacy names (db-backup-<unixtime>.db.gz) have generation 0, so any gen'd backup
// is always considered newer; legacy names remain parseable so retention can delete
// them. Fully unparseable names sort to the end (treated as oldest).
func filterAndSortBackups(objects []model.PrivateObject) []model.PrivateObject {
	var backups []model.PrivateObject
	for _, obj := range objects {
		if strings.HasPrefix(obj.Key, backupPrefix) && strings.HasSuffix(obj.Key, ".db.gz") {
			backups = append(backups, obj)
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		gi, gj := backupGeneration(backups[i].Key), backupGeneration(backups[j].Key)
		if gi != gj {
			return gi > gj
		}
		return backupUnixtime(backups[i].Key) > backupUnixtime(backups[j].Key)
	})

	return backups
}

// backupGeneration extracts the generation counter from a backup object name.
// New names look like "db-backup-gen<gen>-<unixtime>.db.gz" and return <gen>.
// Legacy names "db-backup-<unixtime>.db.gz" (no gen) and unparseable names return 0.
func backupGeneration(key string) int {
	var gen int
	var unix int64
	if n, _ := fmt.Sscanf(key, backupPrefix+genMarker+"%d-%d.db.gz", &gen, &unix); n == 2 {
		return gen
	}
	return 0
}

// backupUnixtime extracts the unix timestamp from a backup object name, handling
// both the new gen'd format and the legacy format. Returns 0 if unparseable.
func backupUnixtime(key string) int64 {
	var gen int
	var unix int64
	if n, _ := fmt.Sscanf(key, backupPrefix+genMarker+"%d-%d.db.gz", &gen, &unix); n == 2 {
		return unix
	}
	if n, _ := fmt.Sscanf(key, backupPrefix+"%d.db.gz", &unix); n == 1 {
		return unix
	}
	return 0
}
