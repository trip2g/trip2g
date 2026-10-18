package mdloader

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestValidateCache(t *testing.T) {
	rules := func(msg string) string {
		return "```jsonnet\n[{level: 'error', msg: '" + msg + "'}]\n```\n"
	}
	note := "---\nvalidate: \"[[rules]]\"\n---\nbody\n"
	other := "---\nvalidate: \"[[other_rules]]\"\n---\nbody\n"
	patch := "---\ntype: frontmatter-patch\ninclude: [\"**\"]\n---\n```jsonnet\n{ x: 1 }\n```\n"

	cache := NewValidateCache()
	load := func(sources map[string]string) []model.NoteWarning {
		t.Helper()
		var srcs []SourceFile
		for p, c := range sources {
			srcs = append(srcs, SourceFile{Path: p, Content: []byte(c)})
		}
		log := logger.TestLogger{}
		nvs, err := Load(Options{Sources: srcs, Log: &log, ValidateCache: cache})
		require.NoError(t, err)
		return nvs.PathMap["note.md"].Warnings
	}
	sentinel := []model.NoteWarning{{Level: model.NoteWarningCritical, Message: "from cache"}}
	plant := func() {
		cache.mu.Lock()
		e := cache.entries["note.md"]
		e.warnings = sentinel
		cache.entries["note.md"] = e
		cache.mu.Unlock()
	}

	sources := map[string]string{
		"rules.md": rules("one"), "note.md": note,
		"other_rules.md": rules("other"), "other.md": other,
	}
	require.Equal(t, "validate rules.md: one", load(sources)[0].Message)

	plant()
	require.Equal(t, sentinel, load(sources), "unchanged note and validator must hit the cache")

	cache.mu.Lock()
	o := cache.entries["other.md"]
	o.warnings = sentinel
	cache.entries["other.md"] = o
	cache.mu.Unlock()

	sources["rules.md"] = rules("two")
	require.Equal(t, "validate rules.md: two", load(sources)[0].Message, "edited validator must rerun")
	cache.mu.Lock()
	require.Equal(t, sentinel, cache.entries["other.md"].warnings, "a note with another validator must stay cached")
	cache.mu.Unlock()

	plant()
	sources["patch.md"] = patch
	require.Equal(t, "validate rules.md: two", load(sources)[0].Message, "changed patch set must rerun")

	plant()
	sources["note.md"] = note + "more\n"
	require.Equal(t, "validate rules.md: two", load(sources)[0].Message, "edited note must rerun")

	delete(sources, "other.md")
	load(sources)
	cache.mu.Lock()
	_, kept := cache.entries["other.md"]
	cache.mu.Unlock()
	require.False(t, kept, "a deleted note's entry must be dropped")
}
