// Package pagecache holds pre-gzipped anonymous rendered-page responses so the
// READ path can skip both re-render and per-request gzip on repeated anonymous
// reads. It stores only fully assembled, role-uniform responses; the caller is
// responsible for never caching personalized, authenticated, or paywalled
// pages (see internal/case/rendernotepage).
//
// The store is a thread-safe expirable LRU (hashicorp/golang-lru/v2): every
// entry has a TTL after which Get misses, and the least-recently-used entry is
// evicted once the size cap is reached. A whole-cache Purge (Clear) invalidates
// everything on note reload.
package pagecache

import (
	"bytes"
	"compress/gzip"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
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

// DefaultTTL bounds how long an entry is served before a forced re-render. It
// is the self-heal window for inputs that have no version counter today (HTML
// injections, user-space JS/CSS bundle). Content and config changes invalidate
// immediately via Clear / ConfigEpoch, so this only governs those few inputs.
// The LRU enforces it internally: Get returns a miss once the entry is expired.
const DefaultTTL = 30 * time.Second

// defaultMaxEntries caps memory. The live key space is bounded
// (#notes x #hosts x 3 langs); once the cap is reached the LRU evicts the
// least-recently-used entry to admit a new page rather than dropping the new one.
const defaultMaxEntries = 8192

// PageCache is a concurrency-safe store of pre-gzipped page bodies backed by an
// expirable LRU.
type PageCache struct {
	lru *expirable.LRU[Key, []byte]
}

// New returns an empty PageCache with the default TTL and size cap.
func New() *PageCache {
	return newWithTTL(defaultMaxEntries, DefaultTTL)
}

// newWithTTL builds a PageCache with an explicit size cap and TTL. Production
// uses New(); tests use it to drive a short real TTL, since expirable.LRU does
// not expose a clock injection hook.
func newWithTTL(maxEntries int, ttl time.Duration) *PageCache {
	return &PageCache{lru: expirable.NewLRU[Key, []byte](maxEntries, nil, ttl)}
}

// Get returns the pre-gzipped bytes for key when present and within its TTL.
// The underlying LRU is internally locked and enforces the TTL, returning a
// miss for expired entries.
func (pc *PageCache) Get(key Key) ([]byte, bool) {
	return pc.lru.Get(key)
}

// Set stores pre-gzipped bytes for key. Past the size cap the LRU evicts the
// least-recently-used entry to admit the new one.
func (pc *PageCache) Set(key Key, gz []byte) {
	pc.lru.Add(key, gz)
}

// Clear purges every entry at once, invalidating the whole cache. Called on
// note reload, where every page must re-render against the new content.
func (pc *PageCache) Clear() {
	pc.lru.Purge()
}

// Len reports the current number of stored entries (including any expired ones
// not yet reaped by the background cleanup). Intended for tests and metrics.
func (pc *PageCache) Len() int {
	return pc.lru.Len()
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
