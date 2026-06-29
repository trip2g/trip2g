package doclint_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/doclint"
	"trip2g/internal/logger"
)

// TestLint_BrokenWikilink verifies that:
//   - a note with a broken wikilink [[does-not-exist]] is reported,
//   - a clean note with no wikilinks produces no warnings.
func TestLint_BrokenWikilink(t *testing.T) {
	dir := t.TempDir()

	// broken.md: references a non-existent note
	writeFile(t, filepath.Join(dir, "broken.md"), "# Broken\n\n[[does-not-exist]]\n")
	// clean.md: no wikilinks, no warnings expected
	writeFile(t, filepath.Join(dir, "clean.md"), "# Clean\n\nJust plain text, nothing broken.\n")

	var buf strings.Builder
	code, err := doclint.Run(context.Background(), dir, &buf, &logger.DummyLogger{})
	require.NoError(t, err)
	require.Equal(t, 1, code, "exit code must be 1 when warnings exist")

	out := buf.String()
	require.Contains(t, out, "broken.md", "broken.md must appear in output")
	require.NotContains(t, out, "clean.md", "clean.md must not appear in output")
}

// TestLint_CleanVault verifies exit 0 and empty output for a vault with no issues.
func TestLint_CleanVault(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "note.md"), "# Hello\n\nThis note has [[sibling]] link.\n")
	writeFile(t, filepath.Join(dir, "sibling.md"), "# Sibling\n\nSibling content.\n")

	var buf strings.Builder
	code, err := doclint.Run(context.Background(), dir, &buf, &logger.DummyLogger{})
	require.NoError(t, err)
	require.Equal(t, 0, code, "exit code must be 0 when vault is clean")
	require.Empty(t, buf.String(), "output must be empty for a clean vault")
}

// TestLint_CrossLangLeak verifies that a bare wikilink in an en/ note that
// resolves to a ru/ note is flagged as a cross-language leak.
func TestLint_CrossLangLeak(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "en"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "ru"), 0o750))

	// en/article.md links to [[topic]] via bare wikilink
	writeFile(t, filepath.Join(dir, "en", "article.md"), "# Article\n\n[[topic]]\n")
	// Only a ru/topic.md exists — so the bare wikilink resolves to ru/
	writeFile(t, filepath.Join(dir, "ru", "topic.md"), "# Topic RU\n\nRussian content.\n")

	var buf strings.Builder
	code, err := doclint.Run(context.Background(), dir, &buf, &logger.DummyLogger{})
	require.NoError(t, err)
	require.Equal(t, 1, code, "exit code must be 1 when cross-lang leak detected")

	out := buf.String()
	require.Contains(t, out, "cross-language", "cross-language leak must be reported")
	require.Contains(t, out, "en/article.md", "source note path must appear in output")
}

// TestLint_FixtureDir runs the linter over docs/demo/lint/ — the dedicated
// known-bad fixture directory — and asserts that every supported warning
// category is reported and that clean.md produces no output.
//
// The fixture directory lives at ../../docs/demo/lint relative to this package.
// Each sub-test below corresponds to one fixture file / warning category.
//
// Note on broken-image detection (fixture: broken_image.md):
// ![[missing.png]] does NOT currently generate a warning. extractInLinks in
// mdloader/loader.go early-returns for any embed where IsMediaExtension is
// true (line ~485), so the "broken image link" fall-through is never reached
// for ![[*.png]] syntax. This is documented as a known limitation in the
// fixture file itself; no assertion is made here.
func TestLint_FixtureDir(t *testing.T) {
	// CWD during 'go test' is the package directory (internal/doclint/).
	dir := filepath.Join("..", "..", "docs", "demo", "lint")

	var buf strings.Builder
	code, err := doclint.Run(context.Background(), dir, &buf, &logger.DummyLogger{})
	require.NoError(t, err, "Run must not return an error for a valid directory")
	require.Equal(t, 1, code, "exit code must be 1 because fixtures contain known warnings")

	out := buf.String()

	t.Run("broken_wikilink", func(t *testing.T) {
		require.Contains(t, out, "broken_wikilink.md", "broken_wikilink.md must appear in output")
		require.Contains(t, out, "broken link: does-not-exist", "broken link message must appear")
	})

	t.Run("cross_lang_leak", func(t *testing.T) {
		// en/crossleak.md links [[ru_only]]; only ru/ru_only.md exists → definite cross-lang leak.
		require.Contains(t, out, "en/crossleak.md", "crossleak note must appear in output")
		require.Contains(t, out, "cross-language", "cross-language warning must be reported")
	})

	t.Run("ambiguous_wikilink", func(t *testing.T) {
		// ru/foo.md links [[bar]]; both en/bar.md and ru/bar.md exist → ambiguous.
		require.Contains(t, out, "ru/foo.md", "ru/foo.md must appear in output")
		require.Contains(t, out, "ambiguous bare wikilink", "ambiguous wikilink warning must appear")
	})

	t.Run("vault_patch_failure", func(t *testing.T) {
		// _vault_patch_bad.md declares type:frontmatter-patch but has no jsonnet block.
		require.Contains(t, out, "_vault_patch_bad.md", "patch failure note must appear in output")
		require.Contains(t, out, "vault patch:", "vault patch error message must appear")
	})

	t.Run("layout_render_error", func(t *testing.T) {
		// _layouts/bad_layout.html accesses note.NoSuchField → smoke render fails.
		require.Contains(t, out, "bad_layout", "bad layout must appear in output")
		require.Contains(t, out, "smoke render", "smoke render error must be reported")
	})

	t.Run("clean_note_is_silent", func(t *testing.T) {
		require.NotContains(t, out, "clean.md", "clean.md must not appear in output")
	})
}

// TestLint_BaselineRatchet verifies the baseline-ratchet:
//   - Warnings present in a baseline file are grandfathered (exit 0, suppressed from output).
//   - New warnings NOT in the baseline cause exit 1 and appear in output.
//   - GenerateBaseline writes a file whose signatures match the current warnings.
func TestLint_BaselineRatchet(t *testing.T) {
	dir := t.TempDir()

	// A note with one broken wikilink.
	writeFile(t, filepath.Join(dir, "broken.md"), "# Broken\n\n[[does-not-exist]]\n")

	// 1. Run without baseline → exit 1.
	var buf strings.Builder
	code, err := doclint.RunWithOptions(context.Background(), dir, &buf, &logger.DummyLogger{}, doclint.Options{})
	require.NoError(t, err)
	require.Equal(t, 1, code, "must fail before baseline is applied")

	// 2. Generate a baseline from the current warnings.
	baselineFile := filepath.Join(t.TempDir(), "baseline.txt")
	require.NoError(t, doclint.GenerateBaseline(context.Background(), dir, baselineFile, &logger.DummyLogger{}))

	// Baseline file must be non-empty.
	data, readErr := os.ReadFile(baselineFile)
	require.NoError(t, readErr)
	require.NotEmpty(t, data, "baseline file must contain at least one entry")

	// 3. Run with baseline → exit 0 (the existing warning is grandfathered).
	var buf2 strings.Builder
	code, err = doclint.RunWithOptions(context.Background(), dir, &buf2, &logger.DummyLogger{}, doclint.Options{BaselineFile: baselineFile})
	require.NoError(t, err)
	require.Equal(t, 0, code, "grandfathered warning must not cause exit 1")
	require.Empty(t, buf2.String(), "baselined warning must be suppressed from output")

	// 4. Add a NEW broken wikilink not in the baseline.
	writeFile(t, filepath.Join(dir, "new_broken.md"), "# New\n\n[[also-missing]]\n")

	var buf3 strings.Builder
	code, err = doclint.RunWithOptions(context.Background(), dir, &buf3, &logger.DummyLogger{}, doclint.Options{BaselineFile: baselineFile})
	require.NoError(t, err)
	require.Equal(t, 1, code, "new warning must cause exit 1")

	out3 := buf3.String()
	require.Contains(t, out3, "also-missing", "new warning must appear in output")
	require.NotContains(t, out3, "does-not-exist", "baselined warning must not appear in output")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
