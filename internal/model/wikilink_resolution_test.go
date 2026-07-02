package model

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// makeLadderNVS builds a NoteViews from path → lang pairs (lang "" means no
// frontmatter lang). BasenameMap is filled in map order on purpose: the
// resolver must not depend on candidate order.
func makeLadderNVS(notes map[string]string) *NoteViews {
	nvs := NewNoteViews()
	// These tests exercise the scoped ladder; global is the default, so opt in.
	nvs.WikilinkResolution = WikilinkResolutionScoped
	nvs.BasenameMap = make(map[string][]*NoteView)
	for path, lang := range notes {
		nv := &NoteView{Path: path, Permalink: "/" + strings.TrimSuffix(path, ".md"), Lang: lang}
		nvs.PathMap[path] = nv
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".md"))
		nvs.BasenameMap[base] = append(nvs.BasenameMap[base], nv)
	}
	return nvs
}

func TestPickBareCandidateLadder(t *testing.T) {
	t.Run("same folder wins over shallower global candidate", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"Topic.md":          "",
			"en/user/Topic.md":  "",
			"en/user/Source.md": "",
			"ru/user/Topic.md":  "",
		})
		source := nvs.PathMap["en/user/Source.md"]
		got := nvs.ResolveWikilinkTarget(source, "Topic")
		require.NotNil(t, got)
		require.Equal(t, "en/user/Topic.md", got.Path)
	})

	t.Run("same lang via path prefix wins over shallower other-lang candidate", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"ru/Topic.md":          "",
			"en/advanced/Topic.md": "",
			"en/user/Source.md":    "",
		})
		source := nvs.PathMap["en/user/Source.md"]
		got := nvs.ResolveWikilinkTarget(source, "Topic")
		require.NotNil(t, got)
		require.Equal(t, "en/advanced/Topic.md", got.Path, "same-lang candidate must beat shallower other-lang one")
	})

	t.Run("same lang via frontmatter wins regardless of path prefix", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"foo/Topic.md":      "ru",
			"bar/deep/Topic.md": "en",
			"baz/Source.md":     "en",
		})
		source := nvs.PathMap["baz/Source.md"]
		got := nvs.ResolveWikilinkTarget(source, "Topic")
		require.NotNil(t, got)
		require.Equal(t, "bar/deep/Topic.md", got.Path)
	})

	t.Run("bilingual equal-depth pair is deterministic and lang-correct", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"en/Topic.md":    "",
			"ru/Topic.md":    "",
			"ru/Source.md":   "",
			"en/EnSource.md": "",
		})
		for range 20 {
			got := nvs.ResolveWikilinkTarget(nvs.PathMap["ru/Source.md"], "Topic")
			require.NotNil(t, got)
			require.Equal(t, "ru/Topic.md", got.Path)

			got = nvs.ResolveWikilinkTarget(nvs.PathMap["en/EnSource.md"], "Topic")
			require.NotNil(t, got)
			require.Equal(t, "en/Topic.md", got.Path)
		}
	})

	t.Run("no folder or lang match falls back to global shallowest with lexicographic tie-break", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"aaa/Topic.md":   "",
			"bbb/Topic.md":   "",
			"docs/Source.md": "",
		})
		for range 20 {
			got := nvs.ResolveWikilinkTarget(nvs.PathMap["docs/Source.md"], "Topic")
			require.NotNil(t, got)
			require.Equal(t, "aaa/Topic.md", got.Path)
		}
	})

	t.Run("single candidate is unchanged", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"ru/deep/Topic.md": "",
			"en/Source.md":     "",
		})
		got := nvs.ResolveWikilinkTarget(nvs.PathMap["en/Source.md"], "Topic")
		require.NotNil(t, got)
		require.Equal(t, "ru/deep/Topic.md", got.Path, "single candidate must resolve even cross-lang")
	})

	t.Run("explicit path escape hatch is unchanged", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"en/user/Topic.md":  "",
			"ru/user/Topic.md":  "",
			"en/user/Source.md": "",
		})
		source := nvs.PathMap["en/user/Source.md"]

		got := nvs.ResolveWikilinkTarget(source, "ru/user/Topic")
		require.NotNil(t, got)
		require.Equal(t, "ru/user/Topic.md", got.Path)

		got = nvs.ResolveWikilinkTarget(source, "./Topic")
		require.NotNil(t, got)
		require.Equal(t, "en/user/Topic.md", got.Path)
	})

	t.Run("global mode (explicit) uses shallowest-only behavior", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"Topic.md":          "",
			"en/user/Topic.md":  "",
			"en/user/Source.md": "",
		})
		nvs.WikilinkResolution = WikilinkResolutionGlobal
		got := nvs.ResolveWikilinkTarget(nvs.PathMap["en/user/Source.md"], "Topic")
		require.NotNil(t, got)
		require.Equal(t, "Topic.md", got.Path, "global mode must ignore the same-folder candidate")
	})

	t.Run("default (unset) is global: root wins even from a subfolder", func(t *testing.T) {
		nvs := makeLadderNVS(map[string]string{
			"Topic.md":          "",
			"en/user/Topic.md":  "",
			"en/user/Source.md": "",
		})
		nvs.WikilinkResolution = "" // no config → Obsidian-compatible global default
		got := nvs.ResolveWikilinkTarget(nvs.PathMap["en/user/Source.md"], "Topic")
		require.NotNil(t, got)
		require.Equal(t, "Topic.md", got.Path, "default must resolve [[Topic]] to root, not the same-folder note")
	})
}

func TestPathLangPrefix(t *testing.T) {
	require.Equal(t, "en", PathLangPrefix("en/user/doc.md"))
	require.Equal(t, "ru", PathLangPrefix("ru/doc.md"))
	require.Empty(t, PathLangPrefix("doc.md"))
	require.Empty(t, PathLangPrefix("docs/en/doc.md"), "only top-level folders count")
	require.Empty(t, PathLangPrefix("english/doc.md"))
	require.Empty(t, PathLangPrefix("go/doc.md"), "non-language two-letter folders are not langs")
}
