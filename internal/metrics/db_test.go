package metrics

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)")
	require.NoError(t, err)

	// Seed a table so page_count > 0 and WAL has data.
	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		_, err = db.Exec("INSERT INTO t (v) VALUES (?)", "hello")
		require.NoError(t, err)
	}

	return db, path
}

func TestNewDBFileCollector_ThreeMetrics(t *testing.T) {
	db, path := openTestDB(t)
	defer db.Close()

	reg := prometheus.NewRegistry()
	c := NewDBFileCollector(db, path)
	reg.MustRegister(c)

	n := testutil.CollectAndCount(c)
	require.Equal(t, 3, n, "expected exactly 3 metrics (size, freelist, wal)")
}

func TestNewDBFileCollector_SizePositive(t *testing.T) {
	db, path := openTestDB(t)
	defer db.Close()

	reg := prometheus.NewRegistry()
	c := NewDBFileCollector(db, path)
	reg.MustRegister(c)

	mfs, err := reg.Gather()
	require.NoError(t, err)

	var sizeVal float64
	var walVal float64
	var foundSize, foundWAL bool
	for _, mf := range mfs {
		switch mf.GetName() {
		case "trip2g_db_size_bytes":
			foundSize = true
			sizeVal = mf.GetMetric()[0].GetGauge().GetValue()
		case "trip2g_db_wal_bytes":
			foundWAL = true
			walVal = mf.GetMetric()[0].GetGauge().GetValue()
		}
	}

	require.True(t, foundSize, "trip2g_db_size_bytes metric must be present")
	require.True(t, foundWAL, "trip2g_db_wal_bytes metric must be present")
	require.Greater(t, sizeVal, float64(0), "trip2g_db_size_bytes must be > 0")
	require.GreaterOrEqual(t, walVal, float64(0), "trip2g_db_wal_bytes must be >= 0")
}

func TestNewDBFileCollector_WALFileAbsent(t *testing.T) {
	// Open in DELETE journal mode so no -wal file is created.
	dir := t.TempDir()
	path := filepath.Join(dir, "nodelete.db")
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(DELETE)")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE x (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	// Ensure no -wal file exists.
	_, statErr := os.Stat(path + "-wal")
	require.True(t, os.IsNotExist(statErr), "expect no -wal file in DELETE mode")

	c := NewDBFileCollector(db, path)

	// Collect should not panic and should still emit 3 metrics (wal=0).
	n := testutil.CollectAndCount(c)
	require.Equal(t, 3, n)
}

func TestNewDBFileCollector_ClosedDBSkipsGracefully(t *testing.T) {
	db, path := openTestDB(t)

	c := NewDBFileCollector(db, path)

	// Close the DB before collecting — Collect must not panic.
	require.NoError(t, db.Close())

	// CollectAndCount internally catches no panic; if Collect panics the test
	// process crashes, which is itself the assertion.
	n := testutil.CollectAndCount(c)
	// Some or all metrics may be absent on error; just verify no panic.
	require.GreaterOrEqual(t, n, 0)
}
