package frontmatterpatch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileVaultPatches_SingleBlock(t *testing.T) {
	patches, err := CompileVaultPatches(
		"vault/patches.md",
		[]string{"*.md"},
		nil,
		[]string{`{ free: true }`},
		100,
	)

	require.NoError(t, err)
	require.Len(t, patches, 1)

	p := patches[0]
	require.Equal(t, []string{"*.md"}, p.IncludePatterns)
	require.Nil(t, p.ExcludePatterns)
	require.Equal(t, 100, p.Priority)
	require.Equal(t, "vault/patches.md", p.Description)
	require.Equal(t, vaultPatchID("vault/patches.md", 0), p.ID)
}

func TestCompileVaultPatches_MultipleBlocks(t *testing.T) {
	patches, err := CompileVaultPatches(
		"vault/patches.md",
		[]string{"*.md"},
		nil,
		[]string{`{ a: 1 }`, `{ b: 2 }`, `{ c: 3 }`},
		50,
	)

	require.NoError(t, err)
	require.Len(t, patches, 3)

	ids := map[int]bool{}
	for i, p := range patches {
		require.Equal(t, vaultPatchID("vault/patches.md", i), p.ID)
		ids[p.ID] = true
	}
	require.Len(t, ids, 3, "all IDs should be unique")
}

func TestCompileVaultPatches_DeterministicIDs(t *testing.T) {
	path := "some/vault/file.md"
	bodies := []string{`{ x: 1 }`, `{ y: 2 }`}

	first, err := CompileVaultPatches(path, []string{"*.md"}, nil, bodies, 0)
	require.NoError(t, err)

	second, err := CompileVaultPatches(path, []string{"*.md"}, nil, bodies, 0)
	require.NoError(t, err)

	for i := range first {
		require.Equal(t, first[i].ID, second[i].ID, "IDs must be stable across calls")
	}

	// Also verify directly via vaultPatchID
	require.Equal(t, vaultPatchID(path, 0), first[0].ID)
	require.Equal(t, vaultPatchID(path, 1), first[1].ID)
}

func TestCompileVaultPatches_EmptyIncludePatterns(t *testing.T) {
	_, err := CompileVaultPatches("vault/patches.md", nil, nil, []string{`{ a: 1 }`}, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "include patterns required")
}

func TestCompileVaultPatches_InvalidIncludePattern(t *testing.T) {
	_, err := CompileVaultPatches("vault/patches.md", []string{"["}, nil, []string{`{ a: 1 }`}, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "include")
}

func TestCompileVaultPatches_InvalidExcludePattern(t *testing.T) {
	_, err := CompileVaultPatches("vault/patches.md", []string{"*.md"}, []string{"["}, []string{`{ a: 1 }`}, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exclude")
}

func TestCompileVaultPatches_InvalidJsonnet(t *testing.T) {
	_, err := CompileVaultPatches(
		"vault/patches.md",
		[]string{"*.md"},
		nil,
		[]string{`{ a: 1 }`, `{ broken: }`},
		0,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "#1")
}

func TestCompileVaultPatches_EmptyJsonnetBodies(t *testing.T) {
	_, err := CompileVaultPatches("vault/patches.md", []string{"*.md"}, nil, []string{}, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no jsonnet blocks found")
}

func TestCompileVaultPatches_PriorityPassedThrough(t *testing.T) {
	patches, err := CompileVaultPatches(
		"vault/patches.md",
		[]string{"*.md"},
		nil,
		[]string{`{ a: 1 }`, `{ b: 2 }`},
		999,
	)
	require.NoError(t, err)
	for _, p := range patches {
		require.Equal(t, 999, p.Priority)
	}
}

func TestCompileVaultPatches_DescriptionFormat(t *testing.T) {
	path := "vault/patches.md"

	t.Run("single block uses plain path", func(t *testing.T) {
		patches, err := CompileVaultPatches(path, []string{"*.md"}, nil, []string{`{ a: 1 }`}, 0)
		require.NoError(t, err)
		require.Equal(t, path, patches[0].Description)
	})

	t.Run("multiple blocks use path#N format", func(t *testing.T) {
		patches, err := CompileVaultPatches(path, []string{"*.md"}, nil, []string{`{ a: 1 }`, `{ b: 2 }`}, 0)
		require.NoError(t, err)
		require.Equal(t, path+"#0", patches[0].Description)
		require.Equal(t, path+"#1", patches[1].Description)
	})
}

func TestCompileVaultPatches_ExcludePatternsNilAndEmpty(t *testing.T) {
	body := []string{`{ a: 1 }`}
	include := []string{"*.md"}

	t.Run("nil exclude", func(t *testing.T) {
		patches, err := CompileVaultPatches("v.md", include, nil, body, 0)
		require.NoError(t, err)
		require.Len(t, patches, 1)
	})

	t.Run("empty exclude", func(t *testing.T) {
		patches, err := CompileVaultPatches("v.md", include, []string{}, body, 0)
		require.NoError(t, err)
		require.Len(t, patches, 1)
	})
}

func TestCompileVaultPatches_WrappedSource(t *testing.T) {
	body := `{ tag: "hello" }`
	patches, err := CompileVaultPatches("vault/patches.md", []string{"*.md"}, nil, []string{body}, 0)
	require.NoError(t, err)

	expected := WrapSource(body)
	require.Equal(t, expected, patches[0].WrappedSource)
	require.Contains(t, patches[0].WrappedSource, body)
	require.Contains(t, patches[0].WrappedSource, `std.extVar("meta")`)
}

func TestVaultPatchIDAlwaysNegative(t *testing.T) {
	paths := []string{
		"blog/rules.md", "_patches/free.md", "a.md", "very/deep/nested/path.md",
		"日本語.md", "special-chars!@#$.md", "UPPERCASE.MD", "",
	}

	for _, path := range paths {
		for i := range 10 {
			id := vaultPatchID(path, i)
			require.Negative(t, id, "vaultPatchID(%q, %d) = %d, want negative", path, i, id)
		}
	}
}
