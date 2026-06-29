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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
