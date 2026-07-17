package codellmgql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/internal/codellm/codellmgql/model"
)

// toInput converts parsed blocks into assemble inputs (drops index, which the
// assembler does not need — order is positional).
func toInput(blocks []model.MarkdownBlock) []model.MdBlockInput {
	out := make([]model.MdBlockInput, len(blocks))
	for i, b := range blocks {
		switch block := b.(type) {
		case model.MarkdownCodeBlock:
			language := block.Language
			if language == "" {
				language = ""
			}
			out[i] = model.MdBlockInput{Kind: model.BlockKindCode, Language: &language, Content: block.Content}
		case model.MarkdownProseBlock:
			out[i] = model.MdBlockInput{Kind: model.BlockKindProse, Content: block.Content}
		}
	}
	return out
}

func TestParseAssemble_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		md   string
	}{
		{"empty", ""},
		{"prose only", "just some prose\nwith two lines\n"},
		{"single code block", "```python\nprint(1)\n```"},
		{"prose then code", "intro text\n\n```bash\necho hi\n```"},
		{"code then prose", "```bash\necho hi\n```\ntrailing note\n"},
		{"prose code prose", "before\n```python\nx=1\n```\nafter\n"},
		{"two adjacent code blocks", "```bash\necho a\n```\n```bash\necho b\n```"},
		{"code no language", "```\nplain code\n```"},
		{"multiline code", "```python\nimport sys\nfor i in range(3):\n    print(i)\n```\ndone\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			blocks := parseMarkdownBlocks(tc.md)
			got := assembleMarkdownBlocks(toInput(blocks))
			require.Equal(t, tc.md, got, "assemble(parse(md)) must reproduce md")
		})
	}
}

// TestParse_BlockKindsAndOrder asserts blocks are ordered, indexed, and tagged
// with the right kind + language, and that code boundaries come from the shared
// fence parser (the code content excludes the fence markers).
func TestParse_BlockKindsAndOrder(t *testing.T) {
	md := "before\n```python\nx=1\n```\nafter\n"
	blocks := parseMarkdownBlocks(md)
	require.Len(t, blocks, 3)

	prose, ok := blocks[0].(model.MarkdownProseBlock)
	require.True(t, ok)
	require.Equal(t, 0, prose.Index)
	require.Equal(t, "before\n", prose.Content)
	require.Contains(t, prose.HTML, "before")

	code, ok := blocks[1].(model.MarkdownCodeBlock)
	require.True(t, ok)
	require.Equal(t, 1, code.Index)
	require.Equal(t, "python", code.Language)
	require.Equal(t, "x=1\n", code.Content)

	prose, ok = blocks[2].(model.MarkdownProseBlock)
	require.True(t, ok)
	require.Equal(t, 2, prose.Index)
	require.Equal(t, "\nafter\n", prose.Content)
}

func TestParseDocumentSeparatesFrontmatterAndRendersProse(t *testing.T) {
	frontmatter, blocks := parseMarkdownDocument("---\nlang: ru\n---\n\n# Hello\n")
	require.Equal(t, "ru", frontmatter["lang"])
	require.Len(t, blocks, 1)
	prose, ok := blocks[0].(model.MarkdownProseBlock)
	require.True(t, ok)
	require.NotContains(t, prose.Content, "lang: ru")
	require.Contains(t, prose.HTML, "<h1>Hello</h1>")
}
