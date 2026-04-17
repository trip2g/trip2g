package mdloader

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestExtractJsonnetBlocks(t *testing.T) {
	tests := []struct {
		name       string
		markdown   string
		wantBlocks []string
		wantErr    string
	}{
		{
			name:       "single jsonnet block",
			markdown:   "```jsonnet\n{foo: 1}\n```\n",
			wantBlocks: []string{"{foo: 1}"},
		},
		{
			name:       "multiple jsonnet blocks extracted in order",
			markdown:   "```jsonnet\n{a: 1}\n```\n\nsome text\n\n```jsonnet\n{b: 2}\n```\n",
			wantBlocks: []string{"{a: 1}", "{b: 2}"},
		},
		{
			name:     "no code blocks at all",
			markdown: "Just plain text.\n",
			wantErr:  "no jsonnet code blocks found",
		},
		{
			name:     "only non-jsonnet code blocks",
			markdown: "```go\npackage main\n```\n\n```json\n{\"x\": 1}\n```\n",
			wantErr:  "no jsonnet code blocks found",
		},
		{
			name:     "empty jsonnet block skipped, error if all empty",
			markdown: "```jsonnet\n   \n```\n",
			wantErr:  "no jsonnet code blocks found",
		},
		{
			name:       "mixed content: text, other code blocks, jsonnet blocks",
			markdown:   "# Title\n\nSome text.\n\n```go\nfunc main() {}\n```\n\nMore text.\n\n```jsonnet\n{x: 42}\n```\n\n```json\n{}\n```\n",
			wantBlocks: []string{"{x: 42}"},
		},
		{
			name:       "jsonnet block content is trimmed of surrounding whitespace",
			markdown:   "```jsonnet\n\n  {trimmed: true}\n\n```\n",
			wantBlocks: []string{"{trimmed: true}"},
		},
		{
			name:       "multi-line jsonnet block preserves internal structure",
			markdown:   "```jsonnet\n{\n  a: 1,\n  b: 2,\n  c: {\n    d: 3,\n  },\n}\n```\n",
			wantBlocks: []string{"{\n  a: 1,\n  b: 2,\n  c: {\n    d: 3,\n  },\n}"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.markdown)
			md := goldmark.New()
			doc := md.Parser().Parse(text.NewReader(source))

			blocks, err := extractJsonnetBlocks(doc, source)

			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				require.Nil(t, blocks)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantBlocks, blocks)
			}
		})
	}
}
