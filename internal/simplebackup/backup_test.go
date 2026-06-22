package simplebackup_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"trip2g/internal/logger"
	"trip2g/internal/miniostorage"
	"trip2g/internal/model"
	"trip2g/internal/simplebackup"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg simplebackup_test . Env

type Env interface {
	Logger() logger.Logger
	DB() *sql.DB
	ListPrivateObjects(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error)
	DeletePrivateObject(ctx context.Context, objectID string) error
	PutPrivateObject(ctx context.Context, reader io.Reader, objectID string) error
	GetPrivateObject(ctx context.Context, objectID string) (io.ReadCloser, error)
}

// createTestDB creates a real SQLite database with test data and returns its path.
func createTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	_, err = db.Exec("CREATE TABLE test_data (id INTEGER PRIMARY KEY, val TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO test_data (val) VALUES ('hello'), ('world')")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	return dbPath
}

// createTestDBWithGen creates a real SQLite database whose PRAGMA user_version
// (the backup generation counter) is set to gen, and returns its path.
func createTestDBWithGen(t *testing.T, gen int) string {
	t.Helper()
	dbPath := createTestDB(t)

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf("PRAGMA user_version = %d", gen))
	require.NoError(t, err)
	require.NoError(t, db.Close())

	return dbPath
}

// gzipFile reads the file at path and returns its gzipped bytes.
func gzipFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write(data)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

// openTestDB opens a SQLite DB and returns it (caller must close).
func openTestDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	return db
}

// makeGzippedSQLite creates a valid gzipped SQLite database and returns its bytes.
func makeGzippedSQLite(t *testing.T) []byte {
	t.Helper()
	dbPath := createTestDB(t)

	data, err := os.ReadFile(dbPath)
	require.NoError(t, err)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write(data)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

// newEnv builds an EnvMock with DummyLogger and the given DB.
func newEnv(db *sql.DB) *EnvMock {
	return &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		DBFunc:     func() *sql.DB { return db },
	}
}

// TestPerformBackup_Success verifies a successful backup uploads gzipped SQLite and calls retention.
func TestPerformBackup_Success(t *testing.T) {
	dbPath := createTestDB(t)
	db := openTestDB(t, dbPath)
	t.Cleanup(func() { db.Close() })

	var uploaded bytes.Buffer
	env := newEnv(db)
	env.PutPrivateObjectFunc = func(ctx context.Context, r io.Reader, objectID string) error {
		_, err := io.Copy(&uploaded, r)
		return err
	}
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return nil, nil // no existing backups
	}
	env.DeletePrivateObjectFunc = func(ctx context.Context, objectID string) error {
		return nil
	}

	mgr := simplebackup.New(env, dbPath)
	err := mgr.PerformBackup(context.Background())
	require.NoError(t, err)

	// Verify PutPrivateObject called once with correct name pattern.
	calls := env.PutPrivateObjectCalls()
	require.Len(t, calls, 1)
	require.True(t, strings.HasPrefix(calls[0].ObjectID, "db-backup-gen"), "object name should start with db-backup-gen")
	require.True(t, strings.HasSuffix(calls[0].ObjectID, ".db.gz"), "object name should end with .db.gz")
	// No existing backups → first generation is 1.
	require.True(t, strings.HasPrefix(calls[0].ObjectID, "db-backup-gen1-"), "first backup should be generation 1, got %s", calls[0].ObjectID)

	// Verify uploaded data is valid gzip containing valid SQLite.
	gr, err := gzip.NewReader(&uploaded)
	require.NoError(t, err, "uploaded data should be valid gzip")
	var sqliteData bytes.Buffer
	_, err = io.Copy(&sqliteData, gr)
	require.NoError(t, err)
	require.NoError(t, gr.Close())
	// SQLite files start with "SQLite format 3".
	require.True(t, bytes.HasPrefix(sqliteData.Bytes(), []byte("SQLite format 3")), "decompressed data should be SQLite")
}

// TestPerformBackup_GenerationIsMaxPlusOne verifies the new generation is
// (max generation across existing S3 backups) + 1, that legacy names count as
// gen 0, and that the snapshot carries the new generation in PRAGMA user_version.
func TestPerformBackup_GenerationIsMaxPlusOne(t *testing.T) {
	dbPath := createTestDB(t)
	db := openTestDB(t, dbPath)
	t.Cleanup(func() { db.Close() })

	existing := []model.PrivateObject{
		{Key: "db-backup-1650000000.db.gz", Size: 100},      // legacy → gen 0
		{Key: "db-backup-gen5-1690000000.db.gz", Size: 100}, // gen 5 (max)
		{Key: "db-backup-gen2-1680000000.db.gz", Size: 100}, // gen 2
	}

	var uploaded bytes.Buffer
	env := newEnv(db)
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return existing, nil
	}
	env.PutPrivateObjectFunc = func(ctx context.Context, r io.Reader, objectID string) error {
		_, err := io.Copy(&uploaded, r)
		return err
	}
	env.DeletePrivateObjectFunc = func(ctx context.Context, objectID string) error { return nil }

	mgr := simplebackup.New(env, dbPath)
	require.NoError(t, mgr.PerformBackup(context.Background()))

	calls := env.PutPrivateObjectCalls()
	require.Len(t, calls, 1)
	require.True(t, strings.HasPrefix(calls[0].ObjectID, "db-backup-gen6-"),
		"new generation should be max(5)+1=6, got %s", calls[0].ObjectID)

	// The uploaded snapshot must carry user_version = 6.
	gr, err := gzip.NewReader(&uploaded)
	require.NoError(t, err)
	var raw bytes.Buffer
	_, err = io.Copy(&raw, gr)
	require.NoError(t, err)
	require.NoError(t, gr.Close())

	snapPath := t.TempDir() + "/snapshot.db"
	require.NoError(t, os.WriteFile(snapPath, raw.Bytes(), 0600))
	gen, err := readUserVersionForTest(t, snapPath)
	require.NoError(t, err)
	require.Equal(t, 6, gen, "snapshot should carry the new generation in user_version")
}

// TestPerformBackup_ConcurrentBlocked verifies that a second backup returns "already in progress".
func TestPerformBackup_ConcurrentBlocked(t *testing.T) {
	dbPath := createTestDB(t)
	db := openTestDB(t, dbPath)
	t.Cleanup(func() { db.Close() })

	started := make(chan struct{})
	unblock := make(chan struct{})

	env := newEnv(db)
	env.PutPrivateObjectFunc = func(ctx context.Context, r io.Reader, objectID string) error {
		close(started)
		<-unblock
		_, _ = io.Copy(io.Discard, r)
		return nil
	}
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return nil, nil
	}
	env.DeletePrivateObjectFunc = func(ctx context.Context, objectID string) error { return nil }

	mgr := simplebackup.New(env, dbPath)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = mgr.PerformBackup(context.Background())
	}()

	<-started // first backup is running

	err := mgr.PerformBackup(context.Background())
	require.ErrorContains(t, err, "backup already in progress")

	close(unblock)
	wg.Wait()
}

// TestPerformBackup_NilDB verifies error when DB is nil.
func TestPerformBackup_NilDB(t *testing.T) {
	dir := t.TempDir()
	env := newEnv(nil) // nil DB
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return nil, nil
	}
	env.DeletePrivateObjectFunc = func(ctx context.Context, objectID string) error { return nil }
	env.PutPrivateObjectFunc = func(ctx context.Context, r io.Reader, objectID string) error { return nil }

	mgr := simplebackup.New(env, dir+"/db.sqlite")
	err := mgr.PerformBackup(context.Background())
	require.ErrorContains(t, err, "database connection is nil")
}

// TestPerformBackup_UploadFails verifies error is propagated when upload fails.
func TestPerformBackup_UploadFails(t *testing.T) {
	dbPath := createTestDB(t)
	db := openTestDB(t, dbPath)
	t.Cleanup(func() { db.Close() })

	env := newEnv(db)
	env.PutPrivateObjectFunc = func(ctx context.Context, r io.Reader, objectID string) error {
		_, _ = io.Copy(io.Discard, r) // drain to avoid goroutine leak
		return errors.New("upload failed: network error")
	}
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return nil, nil
	}
	env.DeletePrivateObjectFunc = func(ctx context.Context, objectID string) error { return nil }

	mgr := simplebackup.New(env, dbPath)
	err := mgr.PerformBackup(context.Background())
	require.ErrorContains(t, err, "upload failed")
}

// TestPerformBackup_RetentionDeletesOldest verifies that oldest backups are deleted when over retentionCount.
// enforceRetention is called after upload, and the mock list contains 5 existing backups.
// filterAndSortBackups gets 5 items, keeps 3 newest → deletes 2 oldest.
func TestPerformBackup_RetentionDeletesOldest(t *testing.T) {
	dbPath := createTestDB(t)
	db := openTestDB(t, dbPath)
	t.Cleanup(func() { db.Close() })

	now := time.Now().Unix()
	// 5 existing backups, sorted newest-first by timestamp in filename.
	existingBackups := []model.PrivateObject{
		{Key: fmt.Sprintf("db-backup-%d.db.gz", now-1000), Size: 100},
		{Key: fmt.Sprintf("db-backup-%d.db.gz", now-2000), Size: 100},
		{Key: fmt.Sprintf("db-backup-%d.db.gz", now-3000), Size: 100},
		{Key: fmt.Sprintf("db-backup-%d.db.gz", now-4000), Size: 100}, // should be deleted
		{Key: fmt.Sprintf("db-backup-%d.db.gz", now-5000), Size: 100}, // should be deleted
	}

	env := newEnv(db)
	env.PutPrivateObjectFunc = func(ctx context.Context, r io.Reader, objectID string) error {
		_, _ = io.Copy(io.Discard, r)
		return nil
	}
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return existingBackups, nil
	}
	env.DeletePrivateObjectFunc = func(ctx context.Context, objectID string) error { return nil }

	mgr := simplebackup.New(env, dbPath)
	err := mgr.PerformBackup(context.Background())
	require.NoError(t, err)

	deleteCalls := env.DeletePrivateObjectCalls()
	// 5 items in mock list, retentionCount=3 → delete 2 oldest.
	require.Len(t, deleteCalls, 2, "should delete 2 oldest backups")

	deletedKeys := make(map[string]bool)
	for _, c := range deleteCalls {
		deletedKeys[c.ObjectID] = true
	}
	require.True(t, deletedKeys[fmt.Sprintf("db-backup-%d.db.gz", now-4000)], "4th oldest should be deleted")
	require.True(t, deletedKeys[fmt.Sprintf("db-backup-%d.db.gz", now-5000)], "5th oldest should be deleted")
}

// TestPerformBackup_RetentionNoDeleteWhenUnderLimit verifies no deletion when at or under retentionCount.
func TestPerformBackup_RetentionNoDeleteWhenUnderLimit(t *testing.T) {
	dbPath := createTestDB(t)
	db := openTestDB(t, dbPath)
	t.Cleanup(func() { db.Close() })

	env := newEnv(db)
	env.PutPrivateObjectFunc = func(ctx context.Context, r io.Reader, objectID string) error {
		_, _ = io.Copy(io.Discard, r)
		return nil
	}
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return []model.PrivateObject{
			{Key: "db-backup-1000.db.gz", Size: 100},
			{Key: "db-backup-2000.db.gz", Size: 100},
		}, nil
	}
	env.DeletePrivateObjectFunc = func(ctx context.Context, objectID string) error { return nil }

	mgr := simplebackup.New(env, dbPath)
	err := mgr.PerformBackup(context.Background())
	require.NoError(t, err)

	// 2 existing in mock list, retentionCount=3 → nothing deleted.
	require.Empty(t, env.DeletePrivateObjectCalls(), "should not delete when under retention limit")
}

// TestRestoreOnStartup_SkipsWhenLocalNotOlder verifies restore is skipped when the
// local DB generation is >= the latest backup's generation (no download happens).
func TestRestoreOnStartup_SkipsWhenLocalNotOlder(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/existing.db"
	// Local DB at generation 5; latest backup is gen 5 → not newer → skip.
	localPath := createTestDBWithGen(t, 5)
	require.NoError(t, os.Rename(localPath, dbPath))

	env := newEnv(nil)
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return []model.PrivateObject{
			{Key: "db-backup-gen5-1700000000.db.gz", Size: 100},
			{Key: "db-backup-gen4-1690000000.db.gz", Size: 100},
		}, nil
	}
	env.GetPrivateObjectFunc = func(ctx context.Context, objectID string) (io.ReadCloser, error) {
		t.Fatal("GetPrivateObject should not be called when local DB is not older")
		return nil, nil
	}

	mgr := simplebackup.New(env, dbPath)
	err := mgr.RestoreOnStartup(context.Background())
	require.NoError(t, err)

	require.Empty(t, env.GetPrivateObjectCalls(), "no download when local DB is not older")
}

// TestRestoreOnStartup_RestoresWhenBackupNewer verifies restore fires (overwriting
// the existing local DB) when the latest backup carries a newer generation.
func TestRestoreOnStartup_RestoresWhenBackupNewer(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/existing.db"
	// Local DB at generation 3 (stale).
	localPath := createTestDBWithGen(t, 3)
	require.NoError(t, os.Rename(localPath, dbPath))

	// Backup snapshot at generation 7.
	backupSrc := createTestDBWithGen(t, 7)
	gzipped := gzipFile(t, backupSrc)

	env := newEnv(nil)
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return []model.PrivateObject{
			{Key: "db-backup-gen7-1700000000.db.gz", Size: int64(len(gzipped))},
			{Key: "db-backup-gen3-1690000000.db.gz", Size: 100},
		}, nil
	}
	env.GetPrivateObjectFunc = func(ctx context.Context, objectID string) (io.ReadCloser, error) {
		require.Equal(t, "db-backup-gen7-1700000000.db.gz", objectID)
		return io.NopCloser(bytes.NewReader(gzipped)), nil
	}

	mgr := simplebackup.New(env, dbPath)
	err := mgr.RestoreOnStartup(context.Background())
	require.NoError(t, err)

	require.Len(t, env.GetPrivateObjectCalls(), 1, "should download the newer backup")

	// The restored DB should carry generation 7.
	gen, err := readUserVersionForTest(t, dbPath)
	require.NoError(t, err)
	require.Equal(t, 7, gen, "local DB should be overwritten with the newer generation")
}

// readUserVersionForTest reads PRAGMA user_version from a SQLite file.
func readUserVersionForTest(t *testing.T, path string) (int, error) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	var gen int
	err = db.QueryRow("PRAGMA user_version").Scan(&gen)
	return gen, err
}

// TestRestoreOnStartup_NoBackups verifies a clean start when no backup files exist.
func TestRestoreOnStartup_NoBackups(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/missing.db" // does not exist

	env := newEnv(nil)
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return []model.PrivateObject{}, nil
	}
	env.GetPrivateObjectFunc = func(ctx context.Context, objectID string) (io.ReadCloser, error) {
		t.Fatal("GetPrivateObject should not be called when no backups exist")
		return nil, nil
	}

	mgr := simplebackup.New(env, dbPath)
	err := mgr.RestoreOnStartup(context.Background())
	require.NoError(t, err)

	_, statErr := os.Stat(dbPath)
	require.True(t, os.IsNotExist(statErr), "DB file should not be created when no backups exist")
	require.Empty(t, env.GetPrivateObjectCalls())
}

// TestRestoreOnStartup_Success verifies restore writes a valid DB with original data.
func TestRestoreOnStartup_Success(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/restore-target.db" // does not exist

	gzipped := makeGzippedSQLite(t)

	env := newEnv(nil)
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return []model.PrivateObject{
			{Key: "db-backup-9999.db.gz", Size: int64(len(gzipped))},
		}, nil
	}
	env.GetPrivateObjectFunc = func(ctx context.Context, objectID string) (io.ReadCloser, error) {
		require.Equal(t, "db-backup-9999.db.gz", objectID)
		return io.NopCloser(bytes.NewReader(gzipped)), nil
	}

	mgr := simplebackup.New(env, dbPath)
	err := mgr.RestoreOnStartup(context.Background())
	require.NoError(t, err)

	_, err = os.Stat(dbPath)
	require.NoError(t, err, "DB file should be created after restore")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count, "restored DB should have 2 rows")
}

// TestRestoreOnStartup_IntegrityCheckFails verifies error when backup is corrupt and DB file is not created.
func TestRestoreOnStartup_IntegrityCheckFails(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/restore-target.db" // does not exist

	// Create gzipped garbage — not a valid SQLite file.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("this is not a sqlite database"))
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	gzipped := buf.Bytes()

	env := newEnv(nil)
	env.ListPrivateObjectsFunc = func(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
		return []model.PrivateObject{
			{Key: "db-backup-9999.db.gz", Size: int64(len(gzipped))},
		}, nil
	}
	env.GetPrivateObjectFunc = func(ctx context.Context, objectID string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(gzipped)), nil
	}

	mgr := simplebackup.New(env, dbPath)
	err = mgr.RestoreOnStartup(context.Background())
	require.ErrorContains(t, err, "integrity check failed")

	_, statErr := os.Stat(dbPath)
	require.True(t, os.IsNotExist(statErr), "DB file should not exist after failed integrity check")
}
