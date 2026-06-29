package frontmatterpatch

import (
	"testing"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/stretchr/testify/require"
)

// load simulates one note-loader Load: it builds a fresh pre-patch meta copy
// (as each Load does) and runs the patch set through the cache, counting how
// many times the Jsonnet-running compute path is taken. ApplyPatches mutates its
// input in place, so each load must start from a fresh copy.
func load(
	c *ResultCache,
	vm *jsonnet.VM,
	patches []CompiledPatch,
	patchSetHash, path string,
	base map[string]interface{},
	computeCalls *int,
) ApplyResult {
	m := deepCopyMeta(base)
	return c.cachedApply(patchSetHash, path, m, func() ApplyResult {
		*computeCalls++
		return ApplyPatches(vm, patches, path, m)
	})
}

func TestResultCache_HitSkipsJsonnet(t *testing.T) {
	vm := NewVM()
	patches := []CompiledPatch{Compile(1, []string{"*.md"}, nil, `meta + { free: true }`, 0, "free")}
	psh := PatchSetHash(patches)
	base := map[string]interface{}{"title": "Test"}

	var calls int
	c := NewResultCache()

	r1 := load(c, vm, patches, psh, "a.md", base, &calls)
	r2 := load(c, vm, patches, psh, "a.md", base, &calls)

	require.Equal(t, 1, calls, "second call must hit the cache and skip the Jsonnet VM")
	require.Equal(t, true, r1.RawMeta["free"])
	require.Equal(t, r1.RawMeta, r2.RawMeta, "hit must return an identical result")
	require.Equal(t, uint64(1), c.Hits())
	require.Equal(t, uint64(1), c.Misses())
}

func TestResultCache_InvalidatePatchSet(t *testing.T) {
	vm := NewVM()
	patchesA := []CompiledPatch{Compile(1, []string{"*.md"}, nil, `meta + { free: true }`, 0, "free")}
	patchesB := []CompiledPatch{Compile(1, []string{"*.md"}, nil, `meta + { free: false }`, 0, "free")}
	pshA := PatchSetHash(patchesA)
	pshB := PatchSetHash(patchesB)
	base := map[string]interface{}{"title": "Test"}

	var calls int
	c := NewResultCache()

	_ = load(c, vm, patchesA, pshA, "a.md", base, &calls)
	require.Equal(t, 1, calls)

	// Same note + same content, but the patch set changed → recompute.
	rB := load(c, vm, patchesB, pshB, "a.md", base, &calls)
	require.Equal(t, 2, calls, "changing the patch set must invalidate the whole cache")
	require.Equal(t, false, rB.RawMeta["free"])

	// New patch set is now cached → a repeat is a hit.
	_ = load(c, vm, patchesB, pshB, "a.md", base, &calls)
	require.Equal(t, 2, calls, "stable patch set must hit again")
}

func TestResultCache_InvalidatePreMeta(t *testing.T) {
	vm := NewVM()
	patches := []CompiledPatch{Compile(1, []string{"*.md"}, nil, `meta + { free: true }`, 0, "free")}
	psh := PatchSetHash(patches)

	var calls int
	c := NewResultCache()

	_ = load(c, vm, patches, psh, "a.md", map[string]interface{}{"title": "One"}, &calls)
	require.Equal(t, 1, calls)

	// Same path, different pre-patch frontmatter → recompute.
	_ = load(c, vm, patches, psh, "a.md", map[string]interface{}{"title": "Two"}, &calls)
	require.Equal(t, 2, calls, "changing the note's frontmatter must invalidate its entry")

	// The latest meta is cached (one entry per path) → repeat is a hit.
	_ = load(c, vm, patches, psh, "a.md", map[string]interface{}{"title": "Two"}, &calls)
	require.Equal(t, 2, calls)
}

func TestResultCache_DifferentPathSeparateEntry(t *testing.T) {
	vm := NewVM()
	patches := []CompiledPatch{Compile(1, []string{"**/*.md"}, nil, `{ filepath: path }`, 0, "path")}
	psh := PatchSetHash(patches)
	base := map[string]interface{}{"title": "Test"}

	var calls int
	c := NewResultCache()

	rA := load(c, vm, patches, psh, "a.md", base, &calls)
	rB := load(c, vm, patches, psh, "b.md", base, &calls)
	require.Equal(t, 2, calls, "distinct paths are distinct entries")
	require.Equal(t, "a.md", rA.RawMeta["filepath"])
	require.Equal(t, "b.md", rB.RawMeta["filepath"])

	// Both paths are cached independently → both hit.
	_ = load(c, vm, patches, psh, "a.md", base, &calls)
	_ = load(c, vm, patches, psh, "b.md", base, &calls)
	require.Equal(t, 2, calls, "both paths must hit on reload")
}

// TestResultCache_HitDeepCopyPreventsCorruption is the no-mutation-corruption
// guarantee: a caller that mutates a returned result (including nested values and
// the applied-patch slice) must not corrupt a subsequent cache hit.
func TestResultCache_HitDeepCopyPreventsCorruption(t *testing.T) {
	vm := NewVM()
	patches := []CompiledPatch{
		Compile(1, []string{"*.md"}, nil, `meta + { nested: { a: "one" }, free: true }`, 0, "nested"),
	}
	psh := PatchSetHash(patches)
	base := map[string]interface{}{"title": "Test"}

	var calls int
	c := NewResultCache()

	r1 := load(c, vm, patches, psh, "a.md", base, &calls)

	// Aggressively corrupt the first result.
	r1.RawMeta["title"] = "HACKED"
	r1.RawMeta["injected"] = true
	if nested, ok := r1.RawMeta["nested"].(map[string]interface{}); ok {
		nested["a"] = "HACKED"
	}
	if len(r1.AppliedPatches) > 0 {
		r1.AppliedPatches[0].Description = "HACKED"
	}

	r2 := load(c, vm, patches, psh, "a.md", base, &calls)

	require.Equal(t, 1, calls, "second call must be a cache hit")
	require.Equal(t, "Test", r2.RawMeta["title"], "cache must not be corrupted")
	require.NotContains(t, r2.RawMeta, "injected")
	require.Equal(t, true, r2.RawMeta["free"])
	nested, ok := r2.RawMeta["nested"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "one", nested["a"], "nested value must survive caller mutation")
	require.Equal(t, "nested", r2.AppliedPatches[0].Description)
}

// TestCachedApply_MatchesUncached proves the cached path (both miss and hit) is
// byte-identical to a direct, uncached ApplyPatches across a range of scenarios.
func TestCachedApply_MatchesUncached(t *testing.T) {
	cases := []struct {
		name    string
		patches []CompiledPatch
		path    string
		meta    map[string]interface{}
	}{
		{
			name:    "simple set",
			patches: []CompiledPatch{Compile(1, []string{"*.md"}, nil, `{ free: true }`, 0, "free")},
			path:    "a.md",
			meta:    map[string]interface{}{"title": "Test"},
		},
		{
			name: "priority chaining",
			patches: []CompiledPatch{
				Compile(1, []string{"*.md"}, nil, `{ category: "blog" }`, 100, "cat"),
				Compile(2, []string{"*.md"}, nil, `meta + { tags: ["general"] }`, 200, "tags"),
			},
			path: "post.md",
			meta: map[string]interface{}{"title": "Test"},
		},
		{
			name:    "no match",
			patches: []CompiledPatch{Compile(1, []string{"blog/**"}, nil, `{ x: 1 }`, 0, "x")},
			path:    "docs/guide.md",
			meta:    map[string]interface{}{"title": "Test"},
		},
		{
			name:    "runtime error warning",
			patches: []CompiledPatch{Compile(1, []string{"*.md"}, nil, `meta.nonexistent.field`, 0, "boom")},
			path:    "a.md",
			meta:    map[string]interface{}{"title": "Test"},
		},
		{
			name:    "nested yaml.v2 map",
			patches: []CompiledPatch{Compile(1, []string{"*.md"}, nil, `meta + { free: true }`, 0, "free")},
			path:    "a.md",
			meta: map[string]interface{}{
				"title": "Test",
				"form": map[interface{}]interface{}{
					"fields": []interface{}{
						map[interface{}]interface{}{"name": "email", "required": true},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM()
			psh := PatchSetHash(tc.patches)

			uncached := ApplyPatches(vm, tc.patches, tc.path, deepCopyMeta(tc.meta))

			c := NewResultCache()
			miss := c.CachedApply(vm, tc.patches, psh, tc.path, deepCopyMeta(tc.meta))
			hit := c.CachedApply(vm, tc.patches, psh, tc.path, deepCopyMeta(tc.meta))

			require.Equal(t, uint64(1), c.Hits())
			require.Equal(t, uncached.RawMeta, miss.RawMeta, "miss must match uncached")
			require.Equal(t, uncached.AppliedPatches, miss.AppliedPatches)
			require.Equal(t, uncached.Warnings, miss.Warnings)
			require.Equal(t, uncached.RawMeta, hit.RawMeta, "hit must match uncached")
			require.Equal(t, uncached.AppliedPatches, hit.AppliedPatches)
			require.Equal(t, uncached.Warnings, hit.Warnings)
		})
	}
}

func TestResultCache_EmptyPatchesUnchanged(t *testing.T) {
	vm := NewVM()
	psh := PatchSetHash(nil)
	base := map[string]interface{}{"title": "Test", "free": true}
	c := NewResultCache()

	r1 := c.CachedApply(vm, nil, psh, "a.md", deepCopyMeta(base))
	r2 := c.CachedApply(vm, nil, psh, "a.md", deepCopyMeta(base))

	require.Empty(t, r1.AppliedPatches)
	require.Equal(t, base, r1.RawMeta, "no patches → meta unchanged")
	require.Equal(t, base, r2.RawMeta)
}

func TestPatchSetHash(t *testing.T) {
	p := func() []CompiledPatch {
		return []CompiledPatch{
			Compile(1, []string{"*.md"}, nil, `{ a: 1 }`, 0, "a"),
			Compile(2, []string{"blog/**"}, []string{"blog/drafts/**"}, `{ b: 2 }`, 10, "b"),
		}
	}

	h1 := PatchSetHash(p())
	h2 := PatchSetHash(p())
	require.Equal(t, h1, h2, "identical sets hash equal")
	require.NotEqual(t, PatchSetHash(p()), PatchSetHash(p()[:1]), "fewer patches → different hash")

	reordered := []CompiledPatch{p()[1], p()[0]}
	require.NotEqual(t, PatchSetHash(p()), PatchSetHash(reordered), "order matters")

	srcChanged := p()
	srcChanged[0] = Compile(1, []string{"*.md"}, nil, `{ a: 2 }`, 0, "a")
	require.NotEqual(t, PatchSetHash(p()), PatchSetHash(srcChanged), "source change → different hash")

	descChanged := p()
	descChanged[0] = Compile(1, []string{"*.md"}, nil, `{ a: 1 }`, 0, "different")
	require.NotEqual(t, PatchSetHash(p()), PatchSetHash(descChanged), "description change → different hash")

	patternChanged := p()
	patternChanged[0] = Compile(1, []string{"docs/**"}, nil, `{ a: 1 }`, 0, "a")
	require.NotEqual(t, PatchSetHash(p()), PatchSetHash(patternChanged), "include change → different hash")

	require.NotEmpty(t, PatchSetHash(nil), "nil set still hashes")
}

func benchPatches() []CompiledPatch {
	return []CompiledPatch{
		Compile(1, []string{"**/*.md"}, nil, `meta + { layout: "post" }`, 0, "layout"),
		Compile(2, []string{"blog/**"}, nil, `meta + { section: "blog", free: true }`, 10, "blog section"),
		Compile(3, []string{"**/*.md"}, nil,
			`if std.objectHas(meta, "draft") && meta.draft then { status: "draft" } else { status: "published" }`,
			20, "status"),
	}
}

func benchMeta() map[string]interface{} {
	return map[string]interface{}{
		"title":  "A Realistic Post Title",
		"tags":   []interface{}{"go", "perf", "caching"},
		"date":   "2026-06-29",
		"author": "alexes",
	}
}

func BenchmarkApplyFrontmatterPatches_Cold(b *testing.B) {
	vm := NewVM()
	patches := benchPatches()
	meta := benchMeta()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = ApplyPatches(vm, patches, "blog/2026/post.md", deepCopyMeta(meta))
	}
}

func BenchmarkApplyFrontmatterPatches_CacheHit(b *testing.B) {
	vm := NewVM()
	patches := benchPatches()
	psh := PatchSetHash(patches)
	meta := benchMeta()
	c := NewResultCache()
	_ = c.CachedApply(vm, patches, psh, "blog/2026/post.md", deepCopyMeta(meta)) // warm
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = c.CachedApply(vm, patches, psh, "blog/2026/post.md", meta)
	}
}
