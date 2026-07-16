package frontmatterpatch

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"

	jsonnet "github.com/google/go-jsonnet"

	"trip2g/internal/yamlutil"
)

// ResultCache memoizes ApplyPatches results across note-loader reloads.
//
// ApplyPatches is a pure, deterministic function of (patches, path, pre-patch
// rawMeta): it only calls MatchPath (a pure glob match) and Evaluate (Jsonnet on
// a sandboxed VM with no importer, native funcs, time, or randomness — only the
// meta+path extVars) followed by a shallow merge. So its result can be safely
// memoized: re-running the Jsonnet VM per note on every reload is pure waste.
//
// The cache stores one entry per note path, validated by two hashes:
//
//   - patchSetHash: a hash of the whole compiled patch set. When ANY patch
//     changes (admin- OR note-defined → the compiled set changes) the hash
//     changes and the entire cache is dropped, so every note recomputes once.
//     This is the cheap, correct invalidation: no need to track which note is
//     affected by which patch.
//   - preMetaHash: a hash of the note's pre-patch rawMeta. Changes only when the
//     note's own frontmatter changes, invalidating just that one entry.
//
// Keeping a single entry per path bounds memory to the note count and
// auto-evicts the stale meta of edited notes.
//
// ResultCache is safe for concurrent use.
type ResultCache struct {
	mu           sync.Mutex
	patchSetHash string
	entries      map[string]cacheEntry

	hits   uint64
	misses uint64
}

type cacheEntry struct {
	preMetaHash string
	result      ApplyResult
}

// NewResultCache returns an empty result cache.
func NewResultCache() *ResultCache {
	return &ResultCache{entries: map[string]cacheEntry{}}
}

// CachedApply returns the patched result for (path, rawMeta) under the compiled
// patch set identified by patchSetHash, memoizing the result across calls. On a
// cache hit it does NOT invoke the Jsonnet VM.
//
// rawMeta must be the PRE-patch frontmatter. On a miss it is passed straight to
// ApplyPatches, which mutates it in place and returns it as the result — matching
// the uncached behavior exactly. On a hit, a private deep copy is returned, so the
// caller may freely mutate the result without corrupting the cache.
func (c *ResultCache) CachedApply(
	vm *jsonnet.VM,
	patches []CompiledPatch,
	patchSetHash, path string,
	rawMeta map[string]interface{},
) ApplyResult {
	return c.cachedApply(patchSetHash, path, rawMeta, func() ApplyResult {
		return ApplyPatches(vm, patches, path, rawMeta)
	})
}

// cachedApply is the cache core with the (Jsonnet-running) compute step injected
// so tests can spy on whether the compute path is taken on a hit.
func (c *ResultCache) cachedApply(
	patchSetHash, path string,
	rawMeta map[string]interface{},
	compute func() ApplyResult,
) ApplyResult {
	preMetaHash := hashMeta(rawMeta)

	c.mu.Lock()
	if c.patchSetHash != patchSetHash {
		// The patch set changed → the whole cache is invalid.
		c.patchSetHash = patchSetHash
		c.entries = make(map[string]cacheEntry, len(c.entries))
	}
	if e, ok := c.entries[path]; ok && e.preMetaHash == preMetaHash {
		hit := cloneResult(e.result)
		c.hits++
		c.mu.Unlock()
		return hit
	}
	c.misses++
	c.mu.Unlock()

	// Compute outside the lock: ApplyPatches uses the shared VM and is the
	// expensive part. A concurrent miss for the same path may compute twice,
	// which is wasteful but harmless (the function is pure).
	result := compute()

	c.mu.Lock()
	// Re-check: a concurrent reload may have reset the cache for a new patch set.
	if c.patchSetHash == patchSetHash {
		c.entries[path] = cacheEntry{preMetaHash: preMetaHash, result: cloneResult(result)}
	}
	c.mu.Unlock()

	return result
}

// Hits returns the number of cache hits served so far. For observability/tests.
func (c *ResultCache) Hits() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits
}

// Misses returns the number of cache misses (Jsonnet recomputes) so far.
func (c *ResultCache) Misses() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.misses
}

// PatchSetHash returns a stable hash identifying a compiled patch set. Two sets
// hash equal iff they would apply identically: same patches in the same order,
// with the same match patterns, priority, source, and description (every field
// that can influence an ApplyResult). Compute once per Load.
func PatchSetHash(patches []CompiledPatch) string {
	h := sha256.New()
	var num [8]byte
	writeStr := func(s string) {
		binary.BigEndian.PutUint64(num[:], uint64(len(s)))
		_, _ = h.Write(num[:])
		_, _ = h.Write([]byte(s))
	}
	writeStrings := func(ss []string) {
		writeStr(strconv.Itoa(len(ss)))
		for _, s := range ss {
			writeStr(s)
		}
	}

	writeStr(strconv.Itoa(len(patches)))
	for _, p := range patches {
		writeStr(strconv.Itoa(p.ID))
		writeStr(strconv.Itoa(p.Priority))
		writeStrings(p.IncludePatterns)
		writeStrings(p.ExcludePatterns)
		writeStr(p.WrappedSource)
		writeStr(p.Description)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// hashMeta returns a deterministic hash of pre-patch frontmatter. It reuses the
// canonical JSON form that Evaluate marshals (encoding/json sorts map keys), so
// two maps with equal content hash equal regardless of iteration order.
func hashMeta(rawMeta map[string]interface{}) string {
	b, err := json.Marshal(yamlutil.Normalize(rawMeta))
	if err != nil {
		// Unmarshalable meta: return a value that never matches a stored entry,
		// forcing a correct (uncached) recompute. Practically unreachable, since
		// yamlutil.Normalize produces JSON-marshalable values.
		return "\x00unhashable"
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// cloneResult returns a deep copy of an ApplyResult that shares no mutable state
// with the original, so a cached snapshot can never be corrupted by a caller.
func cloneResult(r ApplyResult) ApplyResult {
	return ApplyResult{
		RawMeta:        deepCopyMeta(r.RawMeta),
		AppliedPatches: cloneAppliedPatches(r.AppliedPatches),
		Warnings:       cloneStrings(r.Warnings),
	}
}

func cloneAppliedPatches(s []AppliedPatch) []AppliedPatch {
	if s == nil {
		return nil
	}
	out := make([]AppliedPatch, len(s))
	copy(out, s)
	return out
}

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}

// deepCopyMeta returns an independent deep copy of a frontmatter map, preserving
// the original Go value types (unlike a JSON round-trip, which would coerce e.g.
// int to float64). Handles the nested map[interface{}]interface{} /
// map[string]interface{} / []interface{} shapes produced by goldmark-meta.
func deepCopyMeta(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return deepCopyMeta(val)
	case map[interface{}]interface{}:
		out := make(map[interface{}]interface{}, len(val))
		for k, item := range val {
			out[k] = deepCopyValue(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			out[i] = deepCopyValue(item)
		}
		return out
	default:
		// Scalars (string, bool, int, float64, nil, time.Time, …) are immutable.
		return v
	}
}
