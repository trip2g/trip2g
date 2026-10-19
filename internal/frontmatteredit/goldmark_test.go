package frontmatteredit_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"trip2g/internal/frontmatteredit"
)

// The edit is only as good as what trip2g reads back: goldmark-meta, the parser
// that produces a note's meta, must see the keys the edit meant.
func TestApplyIsReadBackByGoldmarkMeta(t *testing.T) {
	tests := []struct {
		name    string
		content string
		set     map[string]any
		unset   []string
		want    map[string]any
	}{
		{
			name:    "note without frontmatter",
			content: "# Title\n\n---\n\ntext\n",
			set:     map[string]any{"status": "done", "tags": []any{"a", "b"}},
			want:    map[string]any{"status": "done", "tags": []any{"a", "b"}},
		},
		{
			name:    "replace, add and remove",
			content: "---\n# c\ntitle: 'T' # t\nweight: 1\ndraft: true\n---\nBody\n",
			set:     map[string]any{"weight": 5, "flag": "yes"},
			unset:   []string{"draft"},
			want:    map[string]any{"title": "T", "weight": 5, "flag": "yes"},
		},
	}

	md := goldmark.New(goldmark.WithExtensions(meta.Meta))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := frontmatteredit.Apply(tt.content, tt.set, tt.unset)
			require.NoError(t, err)

			ctx := parser.NewContext()
			md.Parser().Parse(text.NewReader([]byte(got)), parser.WithContext(ctx))
			require.Equal(t, tt.want, meta.Get(ctx))
		})
	}
}

// requireNothingLost reads both notes the way trip2g does and checks that every
// key the edit did not name kept its value and every set key is there.
func requireNothingLost(t *testing.T, before, after string, set map[string]any, unset []string) {
	t.Helper()

	was := readMeta(t, before)
	now := readMeta(t, after)
	for key, value := range was {
		_, named := set[key]
		if named || slices.Contains(unset, key) {
			continue
		}
		require.Contains(t, now, key)
		require.Equal(t, value, now[key], key)
	}
	for key := range set {
		require.Contains(t, now, key)
	}
	for _, key := range unset {
		require.NotContains(t, now, key)
	}
}

func readMeta(t *testing.T, content string) map[string]any {
	t.Helper()

	source := []byte(strings.TrimPrefix(content, "\uFEFF"))
	ctx := parser.NewContext()
	goldmark.New(goldmark.WithExtensions(meta.Meta)).Parser().Parse(text.NewReader(source), parser.WithContext(ctx))
	got, err := meta.TryGet(ctx)
	require.NoError(t, err)
	return got
}
