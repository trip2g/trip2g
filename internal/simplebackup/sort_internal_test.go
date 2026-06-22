package simplebackup

import (
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// TestBackupGeneration covers parsing the generation counter out of object names,
// including the legacy (no-gen) and unparseable cases.
func TestBackupGeneration(t *testing.T) {
	cases := []struct {
		key  string
		want int
	}{
		{"db-backup-gen7-1700000000.db.gz", 7},
		{"db-backup-gen1-1690000000.db.gz", 1},
		{"db-backup-gen42-0.db.gz", 42},
		{"db-backup-1700000000.db.gz", 0}, // legacy, no gen
		{"db-backup-garbage.db.gz", 0},    // unparseable
		{"unrelated-object.txt", 0},       // not a backup
	}
	for _, c := range cases {
		require.Equal(t, c.want, backupGeneration(c.key), "key=%s", c.key)
	}
}

// TestBackupUnixtime covers parsing the unix timestamp out of both name formats.
func TestBackupUnixtime(t *testing.T) {
	cases := []struct {
		key  string
		want int64
	}{
		{"db-backup-gen7-1700000000.db.gz", 1700000000},
		{"db-backup-1690000000.db.gz", 1690000000}, // legacy
		{"db-backup-garbage.db.gz", 0},             // unparseable
	}
	for _, c := range cases {
		require.Equal(t, c.want, backupUnixtime(c.key), "key=%s", c.key)
	}
}

// TestFilterAndSortBackups verifies ordering with a mix of gen'd and legacy names:
// gen'd backups are always newer than legacy (gen 0); within the same generation
// the higher unixtime wins; legacy names are kept (parseable) so retention can
// still delete them; non-backup objects are filtered out.
func TestFilterAndSortBackups(t *testing.T) {
	input := []model.PrivateObject{
		{Key: "db-backup-1650000000.db.gz"},      // legacy, gen 0
		{Key: "db-backup-gen2-1700000000.db.gz"}, // gen 2
		{Key: "not-a-backup.txt"},                // filtered out
		{Key: "db-backup-gen1-1695000000.db.gz"}, // gen 1
		{Key: "db-backup-1660000000.db.gz"},      // legacy, gen 0, newer ts than the other legacy
		{Key: "db-backup-gen2-1699000000.db.gz"}, // gen 2, older ts than the other gen2
	}

	got := filterAndSortBackups(input)

	gotKeys := make([]string, len(got))
	for i, o := range got {
		gotKeys[i] = o.Key
	}

	want := []string{
		"db-backup-gen2-1700000000.db.gz", // gen 2, newest ts
		"db-backup-gen2-1699000000.db.gz", // gen 2, older ts
		"db-backup-gen1-1695000000.db.gz", // gen 1
		"db-backup-1660000000.db.gz",      // legacy gen 0, newer ts
		"db-backup-1650000000.db.gz",      // legacy gen 0, older ts
	}
	require.Equal(t, want, gotKeys)
}

// TestFilterAndSortBackups_LegacyOnly verifies legacy-only history still sorts
// newest-first by unix timestamp (backward compatibility with pre-gen backups).
func TestFilterAndSortBackups_LegacyOnly(t *testing.T) {
	input := []model.PrivateObject{
		{Key: "db-backup-1000.db.gz"},
		{Key: "db-backup-3000.db.gz"},
		{Key: "db-backup-2000.db.gz"},
	}

	got := filterAndSortBackups(input)
	gotKeys := make([]string, len(got))
	for i, o := range got {
		gotKeys[i] = o.Key
	}
	require.Equal(t, []string{
		"db-backup-3000.db.gz",
		"db-backup-2000.db.gz",
		"db-backup-1000.db.gz",
	}, gotKeys)
}
