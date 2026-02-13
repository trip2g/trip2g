package frontmatterpatch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleExpression(t *testing.T) {
	vm := NewVM()
	patch := Compile(1, []string{"*.md"}, nil, `{ free: true }`, 100, "Set free flag")
	rawMeta := map[string]interface{}{"title": "Test"}

	result := ApplyPatches(vm, []CompiledPatch{patch}, "test.md", rawMeta)

	require.Empty(t, result.Warnings)
	require.Len(t, result.AppliedPatches, 1)
	require.Equal(t, 1, result.AppliedPatches[0].PatchID)
	require.Equal(t, true, result.RawMeta["free"])
	require.Equal(t, "Test", result.RawMeta["title"])
}

func TestExpressionWithMetaPlus(t *testing.T) {
	vm := NewVM()
	patch := Compile(1, []string{"*.md"}, nil, `meta + { author: "system" }`, 100, "Add author")
	rawMeta := map[string]interface{}{"title": "Test"}

	result := ApplyPatches(vm, []CompiledPatch{patch}, "test.md", rawMeta)

	require.Empty(t, result.Warnings)
	require.Len(t, result.AppliedPatches, 1)
	require.Equal(t, "system", result.RawMeta["author"])
	require.Equal(t, "Test", result.RawMeta["title"])
}

func TestConditionalLogic(t *testing.T) {
	vm := NewVM()
	patch := Compile(
		1,
		[]string{"*.md"},
		nil,
		`if std.objectHas(meta, "draft") && meta.draft then { status: "draft" } else { status: "published" }`,
		100,
		"Set status based on draft flag",
	)

	t.Run("with draft flag", func(t *testing.T) {
		rawMeta := map[string]interface{}{"title": "Test", "draft": true}
		result := ApplyPatches(vm, []CompiledPatch{patch}, "test.md", rawMeta)

		require.Empty(t, result.Warnings)
		require.Len(t, result.AppliedPatches, 1)
		require.Equal(t, "draft", result.RawMeta["status"])
	})

	t.Run("without draft flag", func(t *testing.T) {
		rawMeta := map[string]interface{}{"title": "Test"}
		result := ApplyPatches(vm, []CompiledPatch{patch}, "test.md", rawMeta)

		require.Empty(t, result.Warnings)
		require.Len(t, result.AppliedPatches, 1)
		require.Equal(t, "published", result.RawMeta["status"])
	})
}

func TestPriorityChaining(t *testing.T) {
	vm := NewVM()
	patch1 := Compile(1, []string{"*.md"}, nil, `{ category: "blog" }`, 100, "Set category")
	patch2 := Compile(2, []string{"*.md"}, nil, `meta + { tags: ["general"] }`, 200, "Add tags")

	rawMeta := map[string]interface{}{"title": "Test"}
	result := ApplyPatches(vm, []CompiledPatch{patch1, patch2}, "test.md", rawMeta)

	require.Empty(t, result.Warnings)
	require.Len(t, result.AppliedPatches, 2)
	require.Equal(t, "blog", result.RawMeta["category"])
	require.Equal(t, []interface{}{"general"}, result.RawMeta["tags"])
}

func TestIncludeExcludeMatching(t *testing.T) {
	vm := NewVM()
	patch := Compile(
		1,
		[]string{"blog/**/*.md"},
		[]string{"blog/drafts/**"},
		`{ published: true }`,
		100,
		"Mark blog posts as published",
	)

	t.Run("matches include", func(t *testing.T) {
		rawMeta := map[string]interface{}{"title": "Test"}
		result := ApplyPatches(vm, []CompiledPatch{patch}, "blog/2024/post.md", rawMeta)

		require.Empty(t, result.Warnings)
		require.Len(t, result.AppliedPatches, 1)
		require.Equal(t, true, result.RawMeta["published"])
	})

	t.Run("matches exclude", func(t *testing.T) {
		rawMeta := map[string]interface{}{"title": "Test"}
		result := ApplyPatches(vm, []CompiledPatch{patch}, "blog/drafts/post.md", rawMeta)

		require.Empty(t, result.Warnings)
		require.Empty(t, result.AppliedPatches)
		require.NotContains(t, result.RawMeta, "published")
	})

	t.Run("no match", func(t *testing.T) {
		rawMeta := map[string]interface{}{"title": "Test"}
		result := ApplyPatches(vm, []CompiledPatch{patch}, "docs/guide.md", rawMeta)

		require.Empty(t, result.Warnings)
		require.Empty(t, result.AppliedPatches)
		require.NotContains(t, result.RawMeta, "published")
	})
}

func TestRuntimeErrorHandling(t *testing.T) {
	vm := NewVM()
	patch := Compile(1, []string{"*.md"}, nil, `meta.nonexistent.field`, 100, "Access nonexistent field")
	rawMeta := map[string]interface{}{"title": "Test"}

	result := ApplyPatches(vm, []CompiledPatch{patch}, "test.md", rawMeta)

	require.Len(t, result.Warnings, 1)
	require.Contains(t, result.Warnings[0], "Patch 1")
	require.Contains(t, result.Warnings[0], "Access nonexistent field")
	require.Empty(t, result.AppliedPatches)
	require.Equal(t, "Test", result.RawMeta["title"]) // Original meta unchanged
}

func TestJsonnetReturnsNonObject(t *testing.T) {
	vm := NewVM()
	patch := Compile(1, []string{"*.md"}, nil, `"not an object"`, 100, "Return string")
	rawMeta := map[string]interface{}{"title": "Test"}

	result := ApplyPatches(vm, []CompiledPatch{patch}, "test.md", rawMeta)

	require.Len(t, result.Warnings, 1)
	require.Contains(t, result.Warnings[0], "Patch 1")
	require.Empty(t, result.AppliedPatches)
}

func TestEmptyResult(t *testing.T) {
	vm := NewVM()
	patch := Compile(1, []string{"*.md"}, nil, `{}`, 100, "Return empty object")
	rawMeta := map[string]interface{}{"title": "Test"}

	result := ApplyPatches(vm, []CompiledPatch{patch}, "test.md", rawMeta)

	require.Empty(t, result.Warnings)
	require.Len(t, result.AppliedPatches, 1)
	require.Equal(t, "Test", result.RawMeta["title"]) // No changes but patch applied
}

func TestPathVariableAccessible(t *testing.T) {
	vm := NewVM()
	patch := Compile(1, []string{"**/*.md"}, nil, `{ filepath: path }`, 100, "Capture path")
	rawMeta := map[string]interface{}{"title": "Test"}

	result := ApplyPatches(vm, []CompiledPatch{patch}, "docs/guide.md", rawMeta)

	require.Empty(t, result.Warnings)
	require.Len(t, result.AppliedPatches, 1)
	require.Equal(t, "docs/guide.md", result.RawMeta["filepath"])
}

func TestValidateJsonnet(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := ValidateJsonnet(`{ foo: "bar" }`)
		require.NoError(t, err)
	})

	t.Run("valid with meta", func(t *testing.T) {
		err := ValidateJsonnet(`meta + { foo: "bar" }`)
		require.NoError(t, err)
	})

	t.Run("invalid syntax", func(t *testing.T) {
		err := ValidateJsonnet(`{ foo: }`)
		require.Error(t, err)
	})

	t.Run("runtime error", func(t *testing.T) {
		err := ValidateJsonnet(`meta.nonexistent.field`)
		require.Error(t, err)
	})
}

func TestValidatePatterns(t *testing.T) {
	t.Run("valid single", func(t *testing.T) {
		err := ValidatePatterns([]string{"*.md"})
		require.NoError(t, err)
	})

	t.Run("valid multiple", func(t *testing.T) {
		err := ValidatePatterns([]string{"*.md", "blog/**/*.html"})
		require.NoError(t, err)
	})

	t.Run("invalid pattern", func(t *testing.T) {
		err := ValidatePatterns([]string{"["})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid pattern")
	})
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name     string
		include  []string
		exclude  []string
		path     string
		expected bool
	}{
		{
			name:     "simple match",
			include:  []string{"*.md"},
			exclude:  nil,
			path:     "test.md",
			expected: true,
		},
		{
			name:     "no match",
			include:  []string{"*.md"},
			exclude:  nil,
			path:     "test.txt",
			expected: false,
		},
		{
			name:     "exclude takes precedence",
			include:  []string{"**/*.md"},
			exclude:  []string{"drafts/**"},
			path:     "drafts/test.md",
			expected: false,
		},
		{
			name:     "complex glob",
			include:  []string{"blog/{2023,2024}/**/*.md"},
			exclude:  nil,
			path:     "blog/2024/01/post.md",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch := CompiledPatch{
				IncludePatterns: tt.include,
				ExcludePatterns: tt.exclude,
			}
			result := MatchPath(patch, tt.path)
			require.Equal(t, tt.expected, result)
		})
	}
}
