// Package pagecache holds pre-gzipped anonymous rendered-page responses so the
// READ path can skip both re-render and per-request gzip on repeated anonymous
// reads. It stores only fully assembled, role-uniform responses; the caller is
// responsible for never caching personalized, authenticated, or paywalled
// pages (see internal/case/rendernotepage).
//
// Reads are lock-free via an atomic map pointer; writes copy-on-write under a
// mutex. A whole-map swap (Clear) invalidates everything on note reload.
package pagecache

import (
	"bytes"
	"compress/gzip"
	"sync"
	"sync/atomic"
	"time"
)

// Key identifies a cached anonymous rendered page. It is a comparable value
// type used directly as a map key. Every field is part of the cache identity:
//   - Path, Host: the requested URL (note + custom-domain routing / og:url).
//   - NoteVersionID: note content, telegram links, layout sections, lang.
//   - ConfigEpoch: monotonic counter bumped on any site-config change.
//   - UILang: server-embedded ui_lang; only the whitelist {"en","ru",""} is
//     ever cached so this always equals the rendered value.
type Key struct {
	Path          string
	Host          string
	NoteVersionID int64
	ConfigEpoch   uint64
	UILang        string
}

type entry struct {
	gz       []byte
	storedAt time.Time
}

// DefaultTTL bounds how long an entry is served before a forced re-render. It
// is the self-heal window for inputs that have no version counter today (HTML
// injections, user-space JS/CSS bundle). Content and config changes invalidate
// immediately via Clear / ConfigEpoch, so this only governs those few inputs.
const DefaultTTL = 30 * time.Second

// defaultMaxEntries caps memory and copy-on-write cost. The live key space is
// bounded (#notes x #hosts x 3 langs); the cap is a safety valve against
// pathological growth, beyond which new pages are simply served uncached.
const defaultMaxEntries = 8192

// PageCache is a concurrency-safe store of pre-gzipped page bodies.
type PageCache struct {
	m   atomic.Pointer[map[Key]entry]
	mu  sync.Mutex // serializes writers performing copy-on-write
	ttl time.Duration
	max int
	now func() time.Time
}

// New returns an empty PageCache with the default TTL and size cap.
func New() *PageCache {
	pc := &PageCache{
		ttl: DefaultTTL,
		max: defaultMaxEntries,
		now: time.Now,
	}
	empty := map[Key]entry{}
	pc.m.Store(&empty)
	return pc
}

// Get returns the pre-gzipped bytes for key when present and within its TTL.
// Lock-free: a single atomic load of the map pointer on the hot path.
func (pc *PageCache) Get(key Key) ([]byte, bool) {
	mp := pc.m.Load()
	if mp == nil {
		return nil, false
	}
	e, ok := (*mp)[key]
	if !ok {
		return nil, false
	}
	if pc.now().Sub(e.storedAt) >= pc.ttl {
		return nil, false
	}
	return e.gz, true
}

// Set stores pre-gzipped bytes for key. It copies the map on write (so
// concurrent lock-free readers never observe a partially mutated map), pruning
// expired entries during the copy. Past the size cap, a brand-new key is
// dropped (served uncached) rather than growing the map without bound.
func (pc *PageCache) Set(key Key, gz []byte) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	old := pc.m.Load()
	now := pc.now()

	if old != nil && len(*old) >= pc.max {
		if _, exists := (*old)[key]; !exists {
			return
		}
	}

	next := make(map[Key]entry, len(*old)+1)
	for k, v := range *old {
		if now.Sub(v.storedAt) >= pc.ttl {
			continue // prune stale entries while we are copying anyway
		}
		next[k] = v
	}
	next[key] = entry{gz: gz, storedAt: now}
	pc.m.Store(&next)
}

// Clear swaps in a fresh empty map, invalidating every entry at once. Called on
// note reload, where every page must re-render against the new content. It
// takes the writer mutex so it serializes with copy-on-write Set calls: without
// it, a Set whose map snapshot pre-dates the Clear could store the just-cleared
// entries back. (Such resurrected entries are bounded anyway — NoteVersionID and
// ConfigEpoch are in the key, and every entry self-heals within the TTL — but
// serializing makes Clear authoritative.)
func (pc *PageCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	empty := map[Key]entry{}
	pc.m.Store(&empty)
}

// Len reports the current number of stored entries (including any not yet
// pruned expired ones). Intended for tests and metrics.
func (pc *PageCache) Len() int {
	mp := pc.m.Load()
	if mp == nil {
		return 0
	}
	return len(*mp)
}

// Gzip compresses data with gzip at the level fasthttp's CompressHandler uses
// by default (level 6), so cached bytes match the wire format clients already
// receive. The result is a freshly allocated slice owned by the caller.
func Gzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, werr := zw.Write(data); werr != nil {
		return nil, werr
	}
	if cerr := zw.Close(); cerr != nil {
		return nil, cerr
	}
	return buf.Bytes(), nil
}

// setClock overrides the time source; tests use it to drive TTL expiry.
func (pc *PageCache) setClock(now func() time.Time) {
	pc.now = now
}
