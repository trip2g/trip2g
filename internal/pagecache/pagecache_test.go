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
	pc := New()
	now := time.Unix(1000, 0)
	pc.setClock(func() time.Time { return now })

	gz, _ := Gzip([]byte("x"))
	key := Key{Path: "/a"}
	pc.Set(key, gz)

	_, ok := pc.Get(key)
	require.True(t, ok, "fresh entry served")

	now = now.Add(DefaultTTL - time.Millisecond)
	_, ok = pc.Get(key)
	require.True(t, ok, "within TTL still served")

	now = now.Add(2 * time.Millisecond)
	_, ok = pc.Get(key)
	require.False(t, ok, "past TTL not served")
}

func TestSetPrunesExpired(t *testing.T) {
	pc := New()
	now := time.Unix(1000, 0)
	pc.setClock(func() time.Time { return now })

	gz, _ := Gzip([]byte("x"))
	pc.Set(Key{Path: "/old"}, gz)

	now = now.Add(2 * DefaultTTL)
	pc.Set(Key{Path: "/new"}, gz)

	// The stale /old entry is pruned during the copy-on-write of the new Set.
	require.Equal(t, 1, pc.Len())
	_, ok := pc.Get(Key{Path: "/new"})
	require.True(t, ok)
}

func TestSizeCapDropsNewKeys(t *testing.T) {
	pc := New()
	pc.max = 2
	gz, _ := Gzip([]byte("x"))

	pc.Set(Key{Path: "/a"}, gz)
	pc.Set(Key{Path: "/b"}, gz)
	pc.Set(Key{Path: "/c"}, gz) // over cap: dropped
	require.Equal(t, 2, pc.Len())
	_, ok := pc.Get(Key{Path: "/c"})
	require.False(t, ok)

	// Updating an existing key past the cap is still allowed.
	gz2, _ := Gzip([]byte("y"))
	pc.Set(Key{Path: "/a"}, gz2)
	got, ok := pc.Get(Key{Path: "/a"})
	require.True(t, ok)
	require.Equal(t, []byte("y"), gunzip(t, got))
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
