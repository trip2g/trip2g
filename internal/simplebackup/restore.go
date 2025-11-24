package simplebackup

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"trip2g/internal/miniostorage"

	_ "modernc.org/sqlite" // driver for integrity check
)

// RestoreOnStartup restores the DB from S3 storage if local file is missing.
func (m *Manager) RestoreOnStartup(ctx context.Context) error {
	log := m.env.Logger()

	// 1. Check if local DB exists
	if _, err := os.Stat(m.databasePath); err == nil {
		log.Debug("local database exists, skipping restore")
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	log.Info("local database not found, attempting restore")

	// 2. Find latest backup
	objects, err := m.env.ListPrivateObjects(ctx, miniostorage.ListOptions{
		Prefix: backupPrefix,
	})
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(objects) == 0 {
		log.Warn("no backups found, starting with fresh database")
		return nil
	}

	// Sort by LastModified descending (newest first)
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(objects[j].LastModified)
	})

	latest := objects[0]
	log.Info("restoring backup", "key", latest.Key, "size", latest.Size, "modified", latest.LastModified)

	// 3. Download & Decompress
	rc, err := m.env.GetPrivateObject(ctx, latest.Key)
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer rc.Close()

	gzReader, err := gzip.NewReader(rc)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tempRestorePath := m.databasePath + ".restore.tmp"
	outFile, err := os.Create(tempRestorePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		outFile.Close()
		os.Remove(tempRestorePath) // Clean up temp file
	}()

	if _, err := io.Copy(outFile, gzReader); err != nil {
		return fmt.Errorf("failed to write restore file: %w", err)
	}

	outFile.Close() // Close before integrity check

	// 4. Integrity Check
	if err := verifyIntegrity(tempRestorePath); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	// 5. Atomic Move
	if err := os.MkdirAll(filepath.Dir(m.databasePath), 0755); err != nil {
		return fmt.Errorf("failed to create db dir: %w", err)
	}

	if err := os.Rename(tempRestorePath, m.databasePath); err != nil {
		return fmt.Errorf("failed to move restored file: %w", err)
	}

	log.Info("restore successful")
	return nil
}

func verifyIntegrity(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	var result string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return err
	}

	if result != "ok" {
		return fmt.Errorf("result: %s", result)
	}
	return nil
}
