package simplebackup

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"trip2g/internal/miniostorage"

	_ "modernc.org/sqlite" // driver for integrity check
)

// RestoreOnStartup restores the DB from S3 storage when there is no local DB, or
// when the latest S3 backup carries a newer generation than the local DB. The
// generation is stored in the SQLite header via PRAGMA user_version and travels
// inside every snapshot, so the latest backup's generation can be read from its
// object name without downloading it.
// #nosec G110 - Decompression bomb risk is acceptable for trusted backup source.
func (m *Manager) RestoreOnStartup(ctx context.Context) error {
	log := m.env.Logger()

	// 1. Determine whether a local DB already exists.
	localExists := true
	localGen := 0
	if _, err := os.Stat(m.databasePath); err == nil {
		// Read the local generation from its own short-lived read-only connection.
		// This runs pre-DB-init, so we must not rely on m.env.DB(); the connection
		// is closed before any potential overwrite of the file.
		gen, genErr := readUserVersion(m.databasePath)
		if genErr != nil {
			return fmt.Errorf("failed to read local generation: %w", genErr)
		}
		localGen = gen
	} else if os.IsNotExist(err) {
		localExists = false
	} else {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !localExists {
		log.Info("local database not found, attempting restore")
	}

	// 2. Find latest backup
	objects, err := m.env.ListPrivateObjects(ctx, miniostorage.ListOptions{
		Prefix: backupPrefix,
	})
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	backups := filterAndSortBackups(objects)
	if len(backups) == 0 {
		if localExists {
			log.Debug("no backups found, keeping existing local database")
			return nil
		}
		log.Warn("no valid backup files found, starting with fresh database")
		return nil
	}

	latest := backups[0]

	// 3. If a local DB exists, only restore when the latest backup is newer.
	if localExists {
		latestGen := backupGeneration(latest.Key)
		if latestGen <= localGen {
			log.Debug("local database is up to date, skipping restore",
				"localGeneration", localGen, "latestBackupGeneration", latestGen, "key", latest.Key)
			return nil
		}
		log.Info("newer backup available, restoring",
			"localGeneration", localGen, "latestBackupGeneration", latestGen)
	}

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
		_ = outFile.Close()
		_ = os.Remove(tempRestorePath) // Clean up temp file
	}()

	_, copyErr := io.Copy(outFile, gzReader)
	if copyErr != nil {
		return fmt.Errorf("failed to write restore file: %w", copyErr)
	}

	closeErr := outFile.Close() // Close before integrity check
	if closeErr != nil {
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}

	// 4. Integrity Check
	integrityErr := verifyIntegrity(tempRestorePath)
	if integrityErr != nil {
		return fmt.Errorf("integrity check failed: %w", integrityErr)
	}

	// 5. Atomic Move
	mkdirErr := os.MkdirAll(filepath.Dir(m.databasePath), 0750)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create db dir: %w", mkdirErr)
	}

	renameErr := os.Rename(tempRestorePath, m.databasePath)
	if renameErr != nil {
		return fmt.Errorf("failed to move restored file: %w", renameErr)
	}

	log.Info("restore successful")
	return nil
}

// readUserVersion opens the SQLite file read-only on a short-lived connection,
// reads PRAGMA user_version (the backup generation counter), and closes the
// connection so the file can safely be overwritten afterwards.
func readUserVersion(path string) (int, error) {
	// mode=ro prevents opening a writable handle; query_only additionally
	// prevents SQLite from attempting implicit writes (for example a WAL
	// checkpoint) while startup is only inspecting user_version.  Without the
	// latter, a concurrent backup/restore can surface as "attempt to write a
	// readonly database" and abort the whole agent boot.
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var gen int

	err = db.QueryRow("PRAGMA user_version").Scan(&gen)
	if err != nil {
		return 0, err
	}

	return gen, nil
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
