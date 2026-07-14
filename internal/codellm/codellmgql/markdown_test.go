package codellmgql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/codellm/codellmgql/model"
)

// toInput converts parsed blocks into assemble inputs (drops index, which the
// assembler does not need — order is positional).
func toInput(blocks []model.MdBlock) []model.MdBlockInput {
	out := make([]model.MdBlockInput, len(blocks))
	for i, b := range blocks {
		out[i] = model.MdBlockInput{Kind: b.Kind, Language: b.Language, Content: b.Content}
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

	require.Equal(t, 0, blocks[0].Index)
	require.Equal(t, model.BlockKindProse, blocks[0].Kind)
	require.Equal(t, "before\n", blocks[0].Content)
	require.Nil(t, blocks[0].Language)

	require.Equal(t, 1, blocks[1].Index)
	require.Equal(t, model.BlockKindCode, blocks[1].Kind)
	require.NotNil(t, blocks[1].Language)
	require.Equal(t, "python", *blocks[1].Language)
	require.Equal(t, "x=1\n", blocks[1].Content)

	require.Equal(t, 2, blocks[2].Index)
	require.Equal(t, model.BlockKindProse, blocks[2].Kind)
	require.Equal(t, "\nafter\n", blocks[2].Content)
}
