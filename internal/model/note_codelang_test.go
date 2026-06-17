package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractCodeLanguages_CollectsAllFenceLanguages(t *testing.T) {
	content := "```mermaid\ngraph TD;\nA-->B;\n```\n\n" +
		"```go\nfmt.Println(1)\n```\n\n" +
		"```datachart\n{\"data\":{\"source\":\"url\",\"url\":\"http://a\"}}\n```\n"
	n := parseToNoteView(t, content)
	n.extractCodeLanguages()

	require.True(t, n.HasCodeLanguage("mermaid"))
	require.True(t, n.HasCodeLanguage("go"))
	require.True(t, n.HasCodeLanguage("datachart"))
	require.False(t, n.HasCodeLanguage("python"))
}

func TestExtractCodeLanguages_CaseInsensitive(t *testing.T) {
	content := "```Mermaid\ngraph TD;\n```\n"
	n := parseToNoteView(t, content)
	n.extractCodeLanguages()

	require.True(t, n.HasCodeLanguage("mermaid"))
	require.True(t, n.HasCodeLanguage("MERMAID"))
}

func TestExtractCodeLanguages_FirstTokenOnly(t *testing.T) {
	// Obsidian/quicktemplate code fences may carry extra info after the language.
	content := "```js {.line-numbers}\nconst a = 1\n```\n"
	n := parseToNoteView(t, content)
	n.extractCodeLanguages()

	require.True(t, n.HasCodeLanguage("js"))
}

func TestExtractCodeLanguages_NoBlocks(t *testing.T) {
	n := parseToNoteView(t, "# Heading\n\nplain text, `inline code` only.\n")
	n.extractCodeLanguages()

	require.Empty(t, n.CodeLanguages)
	require.False(t, n.HasCodeLanguage("mermaid"))
}

func TestExtractCodeLanguages_NilSafe(t *testing.T) {
	n := &NoteView{}
	require.False(t, n.HasCodeLanguage("mermaid"))
}
