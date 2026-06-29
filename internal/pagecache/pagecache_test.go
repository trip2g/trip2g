package pagecache

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func gunzip(t *testing.T, gz []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	require.NoError(t, err)
	out, err := io.ReadAll(zr)
	require.NoError(t, err)
	return out
}

func TestGzipRoundTrip(t *testing.T) {
	body := []byte("<html><body>hello world</body></html>")
	gz, err := Gzip(body)
	require.NoError(t, err)
	require.Equal(t, body, gunzip(t, gz))
}

func TestGetSet(t *testing.T) {
	pc := New()
	key := Key{Path: "/a", Host: "h", NoteVersionID: 1, UILang: "en"}

	_, ok := pc.Get(key)
	require.False(t, ok, "empty cache must miss")

	gz, err := Gzip([]byte("page-a"))
	require.NoError(t, err)
	pc.Set(key, gz)

	got, ok := pc.Get(key)
	require.True(t, ok)
	require.Equal(t, []byte("page-a"), gunzip(t, got))
}

func TestKeyFieldsAreDistinct(t *testing.T) {
	pc := New()
	base := Key{Path: "/a", Host: "h", NoteVersionID: 1, ConfigEpoch: 0, UILang: "en"}
	variants := []Key{
		base,
		{Path: "/b", Host: "h", NoteVersionID: 1, ConfigEpoch: 0, UILang: "en"},
		{Path: "/a", Host: "h2", NoteVersionID: 1, ConfigEpoch: 0, UILang: "en"},
		{Path: "/a", Host: "h", NoteVersionID: 2, ConfigEpoch: 0, UILang: "en"},
		{Path: "/a", Host: "h", NoteVersionID: 1, ConfigEpoch: 1, UILang: "en"},
		{Path: "/a", Host: "h", NoteVersionID: 1, ConfigEpoch: 0, UILang: "ru"},
	}
	for i, k := range variants {
		gz, _ := Gzip([]byte{byte(i)})
		pc.Set(k, gz)
	}
	require.Equal(t, len(variants), pc.Len(), "each distinct key is its own entry")
}

func TestClearInvalidatesEverything(t *testing.T) {
	pc := New()
	gz, _ := Gzip([]byte("x"))
	pc.Set(Key{Path: "/a"}, gz)
	pc.Set(Key{Path: "/b"}, gz)
	require.Equal(t, 2, pc.Len())

	pc.Clear()
	require.Equal(t, 0, pc.Len())
	_, ok := pc.Get(Key{Path: "/a"})
	require.False(t, ok)
}

func TestTTLExpiry(t *testing.T) {
	// expirable.LRU has no clock injection, so drive a short real TTL + sleep.
	const ttl = 30 * time.Millisecond
	pc := newWithTTL(defaultMaxEntries, ttl)

	gz, _ := Gzip([]byte("x"))
	key := Key{Path: "/a"}
	pc.Set(key, gz)

	_, ok := pc.Get(key)
	require.True(t, ok, "fresh entry served")

	time.Sleep(2 * ttl)
	_, ok = pc.Get(key)
	require.False(t, ok, "past TTL not served")
}

func TestSizeCapEvictsLRU(t *testing.T) {
	// The cap now does LRU eviction (least-recently-used evicted to admit a new
	// key), NOT drop-new.
	pc := newWithTTL(2, DefaultTTL)
	gz, _ := Gzip([]byte("x"))

	pc.Set(Key{Path: "/a"}, gz)
	pc.Set(Key{Path: "/b"}, gz)
	require.Equal(t, 2, pc.Len())

	// Touch /a so /b becomes the least-recently-used entry.
	_, ok := pc.Get(Key{Path: "/a"})
	require.True(t, ok)

	// Adding a new key over the cap evicts the LRU (/b), not the new key.
	pc.Set(Key{Path: "/c"}, gz)
	require.Equal(t, 2, pc.Len(), "cap holds: still max entries")

	_, ok = pc.Get(Key{Path: "/c"})
	require.True(t, ok, "new key is admitted (LRU eviction, not drop-new)")
	_, ok = pc.Get(Key{Path: "/a"})
	require.True(t, ok, "recently-used key retained")
	_, ok = pc.Get(Key{Path: "/b"})
	require.False(t, ok, "least-recently-used key was evicted")
}

func TestConcurrentGetSet(t *testing.T) {
	// Exercises the lock-free read / copy-on-write write paths under -race.
	pc := New()
	gz, _ := Gzip([]byte("x"))
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := range 200 {
				k := Key{Path: "/p", NoteVersionID: int64(j % 16)}
				pc.Set(k, gz)
				pc.Get(k)
				pc.Get(Key{Path: "/p", NoteVersionID: int64(i)})
			}
		}(i)
	}
	wg.Wait()
}
