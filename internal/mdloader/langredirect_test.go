package mdloader_test

import (
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

// TestLangRedirectSymmetricPairDeterministic reproduces the flaky
// "already belongs to another lang group" warning: a symmetric root pair
// (docs/_index en ↔ docs/ru/_index ru) coexists with several same-basename
// _index notes. The root en→ru redirect uses an explicit path; the ru→en
// redirect is a BARE [[_index]], which must resolve to the root en index —
// never to itself (same-folder step) and never to a same-lang sibling.
//
// The test shuffles source order across iterations (source order drives
// PathMap/build order) and asserts: no lang-group warning, a valid symmetric
// pair, and stable LangAlternatives.
func TestLangRedirectSymmetricPairDeterministic(t *testing.T) {
	log := logger.TestLogger{}

	// Distinct orderings to exercise build-order independence.
	orderings := [][]mdloader.SourceFile{
		langRedirectFixture("_index.md", "ru/_index.md"),
		langRedirectFixture("ru/_index.md", "_index.md"),
		langRedirectFixture("en/hub/_index.md", "_index.md"),
	}

	for _, srcs := range orderings {
		for range 20 {
			pages, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
			require.NoError(t, err)

			root := pages.PathMap["_index.md"]
			ru := pages.PathMap["ru/_index.md"]
			require.NotNil(t, root)
			require.NotNil(t, ru)

			for _, w := range root.Warnings {
				require.NotContains(t, w.Message, "already belongs to another lang group", "root must not warn")
				require.NotContains(t, w.Message, "lang_redirect", "root lang_redirect must resolve cleanly: %s", w.Message)
			}
			for _, w := range ru.Warnings {
				require.NotContains(t, w.Message, "already belongs to another lang group", "ru must not warn")
				require.NotContains(t, w.Message, "lang_redirect", "ru lang_redirect must resolve cleanly: %s", w.Message)
			}

			// Symmetric pair: each points at the other, never at itself.
			require.Len(t, root.LangRedirects, 1)
			require.Equal(t, "ru/_index.md", root.LangRedirects[0].Note.Path, "root en redirects to ru root")
			require.Len(t, ru.LangRedirects, 1)
			require.Equal(t, "_index.md", ru.LangRedirects[0].Note.Path, "ru redirects to root en (not itself)")

			// Stable LangAlternatives across both members of the pair.
			require.Equal(t, ru, root.LangAlternatives["ru"])
			require.Equal(t, root, ru.LangAlternatives["en"])
		}
	}
}

// langRedirectFixture returns the six same-basename _index notes with a leading
// pair of the caller's choosing (only ordering changes; content is fixed).
func langRedirectFixture(first, second string) []mdloader.SourceFile {
	byPath := map[string]mdloader.SourceFile{
		"_index.md":         {Path: "_index.md", Content: []byte("---\nlang: en\nlang_redirect: \"[[ru/_index]]\"\n---\nroot en")},
		"ru/_index.md":      {Path: "ru/_index.md", Content: []byte("---\nlang: ru\nlang_redirect: \"[[_index]]\"\n---\nroot ru")},
		"en/user/_index.md": {Path: "en/user/_index.md", Content: []byte("---\nlang: en\nlang_redirect: \"[[ru/user/_index]]\"\n---\nen user")},
		"ru/user/_index.md": {Path: "ru/user/_index.md", Content: []byte("---\nlang: ru\nlang_redirect: \"[[en/user/_index]]\"\n---\nru user")},
		"en/hub/_index.md":  {Path: "en/hub/_index.md", Content: []byte("---\nlang: en\nlang_redirect: \"[[ru/hub/_index]]\"\n---\nen hub")},
		"ru/hub/_index.md":  {Path: "ru/hub/_index.md", Content: []byte("---\nlang: ru\nlang_redirect: \"[[en/hub/_index]]\"\n---\nru hub")},
	}
	order := []string{first, second}
	for p := range byPath {
		if p != first && p != second {
			order = append(order, p)
		}
	}
	out := make([]mdloader.SourceFile, 0, len(order))
	for _, p := range order {
		out = append(out, byPath[p])
	}
	return out
}

// TestLangRedirectBareNeverSelf asserts a bare [[_index]] lang_redirect from a
// note whose only same-folder candidate is itself resolves to the cross-language
// root, not to the note itself.
func TestLangRedirectBareNeverSelf(t *testing.T) {
	log := logger.TestLogger{}
	srcs := langRedirectFixture("ru/_index.md", "_index.md")

	pages, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	require.NoError(t, err)

	ru := pages.PathMap["ru/_index.md"]
	got := pages.ResolveLangRedirectTarget(ru, "_index")
	require.NotNil(t, got)
	require.Equal(t, "_index.md", got.Path)
	require.False(t, strings.HasPrefix(got.Path, "ru/"), "must not resolve to a same-language candidate")
}
