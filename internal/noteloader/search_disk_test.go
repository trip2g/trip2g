package noteloader_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/noteloader"
)

// capturingLogger records what the loader says, which is how these tests tell a
// reused index from a rebuilt one: the counts only show up in the log line.
type capturingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *capturingLogger) record(msg string, kv ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fmt.Sprint(append([]interface{}{msg}, kv...)...))
}

func (l *capturingLogger) Info(msg string, kv ...interface{})  { l.record(msg, kv...) }
func (l *capturingLogger) Error(msg string, kv ...interface{}) { l.record(msg, kv...) }
func (l *capturingLogger) Debug(msg string, kv ...interface{}) { l.record(msg, kv...) }
func (l *capturingLogger) Warn(msg string, kv ...interface{})  { l.record(msg, kv...) }

func (l *capturingLogger) find(t *testing.T, substrings ...string) string {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
outer:
	for _, line := range l.lines {
		for _, sub := range substrings {
			if !contains(line, sub) {
				continue outer
			}
		}
		return line
	}
	return ""
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && stringIndex(haystack, needle) >= 0)
}

func stringIndex(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func note(pathID int64, path, title, body string) noteloader.RawNote {
	return noteloader.RawNote{
		Path:      path,
		PathID:    pathID,
		VersionID: pathID,
		Content:   fmt.Sprintf("---\ntitle: %s\n---\n%s", title, body),
	}
}

// diskLoader builds a loader whose index lives under dir, with its own log.
func diskLoader(t *testing.T, dir string, notes []noteloader.RawNote) (*noteloader.Loader, *capturingLogger) {
	t.Helper()
	log := &capturingLogger{}
	env := makeMinimalEnv(notes, func() bool { return false })
	env.LoggerFunc = func() logger.Logger { return log }

	loader := noteloader.New("test", env, mdloader.Config{})
	loader.SetSearchIndexPath(dir)
	require.NoError(t, loader.Load(context.Background(), noteloader.LoadOptions{}))
	return loader, log
}

// The point of putting the index on disk: a restart reuses it instead of
// re-indexing every note.
func TestSearchIndex_OnDisk_ReusedAfterRestart(t *testing.T) {
	dir := t.TempDir()
	notes := []noteloader.RawNote{
		note(1, "alpha.md", "Alpha", "уникальное слово капибара"),
		note(2, "beta.md", "Beta", "совсем другой текст"),
	}

	first, firstLog := diskLoader(t, dir, notes)
	found, err := first.Search("капибара")
	require.NoError(t, err)
	require.Len(t, found, 1, "the note must be findable right after the first build")
	require.NotEmpty(t, firstLog.find(t, "search index created on disk"))
	require.NoError(t, first.Close())

	second, secondLog := diskLoader(t, dir, notes)
	t.Cleanup(func() { _ = second.Close() })

	require.NotEmpty(t, secondLog.find(t, "search index opened from disk"),
		"the second loader must open the existing index, not create one")
	require.NotEmpty(t, secondLog.find(t, "persisted search index adopted", "adopted2"),
		"both documents must be adopted from disk")
	require.NotEmpty(t, secondLog.find(t, "notes indexed", "indexed0", "skipped2"),
		"unchanged notes must be skipped, not re-indexed")

	found, err = second.Search("капибара")
	require.NoError(t, err)
	require.Len(t, found, 1, "search must work against the reopened index")
}

// The failure mode a persisted index introduces: deletion is detected by
// walking an in-memory map, and that map is empty on a fresh process.
func TestSearchIndex_OnDisk_DropsNotesDeletedWhileDown(t *testing.T) {
	dir := t.TempDir()
	before := []noteloader.RawNote{
		note(1, "alpha.md", "Alpha", "уникальное слово капибара"),
		note(2, "beta.md", "Beta", "второе слово выхухоль"),
	}

	first, _ := diskLoader(t, dir, before)
	require.NoError(t, first.Close())

	// beta.md is gone from the database while nothing was running.
	second, secondLog := diskLoader(t, dir, before[:1])
	t.Cleanup(func() { _ = second.Close() })

	require.NotEmpty(t, secondLog.find(t, "persisted search index adopted", "orphans_removed1"))

	found, err := second.Search("выхухоль")
	require.NoError(t, err)
	require.Empty(t, found, "a note deleted while the process was down must not haunt the index")

	found, err = second.Search("капибара")
	require.NoError(t, err)
	require.Len(t, found, 1, "the surviving note must still be there")
}

// A changed note must be re-indexed even though the index survived the restart.
func TestSearchIndex_OnDisk_ReindexesChangedNote(t *testing.T) {
	dir := t.TempDir()
	first, _ := diskLoader(t, dir, []noteloader.RawNote{note(1, "alpha.md", "Alpha", "старое слово капибара")})
	require.NoError(t, first.Close())

	second, secondLog := diskLoader(t, dir, []noteloader.RawNote{note(1, "alpha.md", "Alpha", "новое слово выхухоль")})
	t.Cleanup(func() { _ = second.Close() })

	require.NotEmpty(t, secondLog.find(t, "notes indexed", "indexed1"), "a changed note must be re-indexed")

	found, err := second.Search("выхухоль")
	require.NoError(t, err)
	require.Len(t, found, 1, "the new text must be searchable")

	found, err = second.Search("капибара")
	require.NoError(t, err)
	require.Empty(t, found, "the old text must be gone")
}

// A schema bump must not leave a stale index serving results from an old mapping.
func TestSearchIndex_OnDisk_RemovesOtherSchemaVersions(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "test", "v0")
	require.NoError(t, os.MkdirAll(stale, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(stale, "marker"), []byte("old"), 0o600))

	loader, _ := diskLoader(t, dir, []noteloader.RawNote{note(1, "alpha.md", "Alpha", "текст")})
	t.Cleanup(func() { _ = loader.Close() })

	_, err := os.Stat(stale)
	require.True(t, os.IsNotExist(err), "an index from another schema version must be deleted")
	_, err = os.Stat(filepath.Join(dir, "test", "v1"))
	require.NoError(t, err, "the current schema version must exist")
}

// Default behaviour is unchanged: no path, no files, index in memory.
func TestSearchIndex_InMemoryByDefault(t *testing.T) {
	dir := t.TempDir()
	env := makeMinimalEnv([]noteloader.RawNote{note(1, "alpha.md", "Alpha", "капибара")}, func() bool { return false })
	loader := noteloader.New("test", env, mdloader.Config{})
	require.NoError(t, loader.Load(context.Background(), noteloader.LoadOptions{}))
	t.Cleanup(func() { _ = loader.Close() })

	found, err := loader.Search("капибара")
	require.NoError(t, err)
	require.Len(t, found, 1)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries, "without a configured path nothing may touch the disk")
}

// An index that cannot be opened must never be deleted: during a zero-downtime
// handoff the old instance still holds it, and "cannot open" means "busy" far
// more often than it means "corrupt".
func TestSearchIndex_OnDisk_UnreadableFallsBackWithoutDeleting(t *testing.T) {
	dir := t.TempDir()
	indexDir := filepath.Join(dir, "test", "v1")
	require.NoError(t, os.MkdirAll(indexDir, 0o750))
	// Not an index, just something bleve will refuse to open.
	require.NoError(t, os.WriteFile(filepath.Join(indexDir, "index_meta.json"), []byte("{"), 0o600))

	loader, log := diskLoader(t, dir, []noteloader.RawNote{note(1, "alpha.md", "Alpha", "капибара")})
	t.Cleanup(func() { _ = loader.Close() })

	require.NotEmpty(t, log.find(t, "falling back to the in-memory index"),
		"the loader must say out loud that it did not get the on-disk index")

	found, err := loader.Search("капибара")
	require.NoError(t, err)
	require.Len(t, found, 1, "search must keep working on the in-memory fallback")

	_, err = os.Stat(filepath.Join(indexDir, "index_meta.json"))
	require.NoError(t, err, "the unreadable index must be left alone, not deleted")
}
