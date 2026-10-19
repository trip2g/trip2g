package frontmatteredit_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/frontmatteredit"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name    string
		content string
		set     map[string]any
		unset   []string
		want    string
	}{
		{
			name:    "absent key is appended after the last key",
			content: "---\ntitle: Hello\n---\nBody\n",
			set:     map[string]any{"status": "done"},
			want:    "---\ntitle: Hello\nstatus: done\n---\nBody\n",
		},
		{
			name:    "present key is replaced in place",
			content: "---\ntitle: Hello\nstatus: draft\ntags: [a]\n---\nBody\n",
			set:     map[string]any{"status": "done"},
			want:    "---\ntitle: Hello\nstatus: done\ntags: [a]\n---\nBody\n",
		},
		{
			name:    "unset removes the key and its block value",
			content: "---\ntitle: Hello\ntags:\n  - a\n  - b\nstatus: draft\n---\nBody\n",
			unset:   []string{"tags"},
			want:    "---\ntitle: Hello\nstatus: draft\n---\nBody\n",
		},
		{
			name:    "unset of an absent key changes nothing",
			content: "---\ntitle: Hello\n---\nBody\n",
			unset:   []string{"missing"},
			want:    "---\ntitle: Hello\n---\nBody\n",
		},
		{
			name:    "unset on a note without frontmatter changes nothing",
			content: "Body\n",
			unset:   []string{"missing"},
			want:    "Body\n",
		},
		{
			name:    "a note without frontmatter gets one and keeps its body",
			content: "# Title\n\n---\n\ntext  \n",
			set:     map[string]any{"status": "done"},
			want:    "---\nstatus: done\n---\n# Title\n\n---\n\ntext  \n",
		},
		{
			name:    "an empty note gets frontmatter",
			content: "",
			set:     map[string]any{"status": "done"},
			want:    "---\nstatus: done\n---\n",
		},
		{
			name:    "empty frontmatter gets the key",
			content: "---\n---\nBody\n",
			set:     map[string]any{"status": "done"},
			want:    "---\nstatus: done\n---\nBody\n",
		},
		{
			name: "comments, quoting and order of other keys survive",
			content: "---\n# leading comment\ntitle: 'Quoted'   # title note\n\n" +
				"# about status\nstatus: draft # was draft\nlist:\n- x\n- y\n# trailing comment\n---\nBody\n",
			set: map[string]any{"status": "done"},
			want: "---\n# leading comment\ntitle: 'Quoted'   # title note\n\n" +
				"# about status\nstatus: done # was draft\nlist:\n- x\n- y\n# trailing comment\n---\nBody\n",
		},
		{
			name:    "new keys are added in sorted order",
			content: "---\ntitle: Hello\n---\n",
			set:     map[string]any{"zeta": 1, "alpha": 2},
			want:    "---\ntitle: Hello\nalpha: 2\nzeta: 1\n---\n",
		},
		{
			name:    "bool value",
			content: "---\nfree: false\n---\n",
			set:     map[string]any{"free": true},
			want:    "---\nfree: true\n---\n",
		},
		{
			name:    "int value from JSON",
			content: "---\nweight: 1\n---\n",
			set:     map[string]any{"weight": json.Number("42")},
			want:    "---\nweight: 42\n---\n",
		},
		{
			name:    "float value decoded by encoding/json",
			content: "---\n---\n",
			set:     map[string]any{"weight": float64(3), "ratio": 0.5},
			want:    "---\nratio: 0.5\nweight: 3\n---\n",
		},
		{
			name:    "string that looks like another type is quoted",
			content: "---\n---\n",
			set:     map[string]any{"version": "1.0", "flag": "true", "empty": ""},
			want:    "---\nempty: \"\"\nflag: \"true\"\nversion: \"1.0\"\n---\n",
		},
		{
			name:    "multiline string becomes a block scalar",
			content: "---\n---\n",
			set:     map[string]any{"summary": "one\ntwo\n"},
			want:    "---\nsummary: |\n  one\n  two\n---\n",
		},
		{
			name:    "list value",
			content: "---\ntitle: Hello\ntags: [old]\n---\n",
			set:     map[string]any{"tags": []any{"a", "b"}},
			want:    "---\ntitle: Hello\ntags:\n  - a\n  - b\n---\n",
		},
		{
			name:    "nested map value",
			content: "---\n---\n",
			set:     map[string]any{"og": map[string]any{"title": "T", "count": 1}},
			want:    "---\nog:\n  count: 1\n  title: T\n---\n",
		},
		{
			name:    "null value",
			content: "---\n---\n",
			set:     map[string]any{"cleared": nil},
			want:    "---\ncleared: null\n---\n",
		},
		{
			name:    "setting the current scalar value keeps its formatting",
			content: "---\ntitle: \"Hello\" # c\n---\n",
			set:     map[string]any{"title": "Hello"},
			want:    "---\ntitle: \"Hello\" # c\n---\n",
		},
		{
			name:    "dotted key is a literal top-level key",
			content: "---\nog:\n  title: T\n---\n",
			set:     map[string]any{"og.title": "X"},
			want:    "---\nog:\n  title: T\nog.title: X\n---\n",
		},
		{
			name:    "set and unset together",
			content: "---\na: 1\nb: 2\nc: 3\n---\n",
			set:     map[string]any{"b": 20, "d": 4},
			unset:   []string{"a", "c"},
			want:    "---\nb: 20\nd: 4\n---\n",
		},
		{
			name:    "replacing the last key keeps trailing blank and comment lines",
			content: "---\na: 1\nb: 2\n\n# end\n---\nBody",
			set:     map[string]any{"b": 3},
			want:    "---\na: 1\nb: 3\n\n# end\n---\nBody",
		},
		{
			name:    "block scalar value is replaced whole",
			content: "---\ndesc: |\n  line one\n  line two\nnext: x\n---\n",
			set:     map[string]any{"desc": "short"},
			want:    "---\ndesc: short\nnext: x\n---\n",
		},
		{
			name:    "CRLF note keeps CRLF",
			content: "---\r\ntitle: Hello\r\n---\r\nBody\r\n",
			set:     map[string]any{"status": "done"},
			want:    "---\r\ntitle: Hello\r\nstatus: done\r\n---\r\nBody\r\n",
		},
		{
			name:    "CRLF note without frontmatter gets CRLF fences",
			content: "Body\r\n",
			set:     map[string]any{"status": "done"},
			want:    "---\r\nstatus: done\r\n---\r\nBody\r\n",
		},
		{
			name:    "byte order mark stays first",
			content: "\uFEFF---\ntitle: Hello\n---\nBody\n",
			set:     map[string]any{"title": "Bye"},
			want:    "\uFEFF---\ntitle: Bye\n---\nBody\n",
		},
		{
			name:    "longer fences are kept",
			content: "-----\ntitle: Hello\n----\nBody\n",
			unset:   []string{"title"},
			want:    "-----\n----\nBody\n",
		},
		{
			name:    "body without trailing newline is untouched",
			content: "---\na: 1\n---\nno newline",
			set:     map[string]any{"a": 2},
			want:    "---\na: 2\n---\nno newline",
		},
		{
			name:    "key that needs quoting",
			content: "---\n---\n",
			set:     map[string]any{"a: b": 1, "true": "x"},
			want:    "---\n'a: b': 1\n\"true\": x\n---\n",
		},
		{
			name:    "line separator inside a quoted value: set the next key",
			content: "---\ntitle: \"a\u2028b\"\nstatus: draft\ntags: [x]\n---\nBody\n",
			set:     map[string]any{"status": "done"},
			want:    "---\ntitle: \"a\u2028b\"\nstatus: done\ntags: [x]\n---\nBody\n",
		},
		{
			name:    "line separator inside a quoted value: unset the next key",
			content: "---\ntitle: \"a\u2028b\"\nstatus: draft\ntags: [x]\n---\nBody\n",
			unset:   []string{"status"},
			want:    "---\ntitle: \"a\u2028b\"\ntags: [x]\n---\nBody\n",
		},
		{
			name:    "paragraph separator inside a quoted value",
			content: "---\ntitle: \"a\u2029b\"\nstatus: draft\ntags: [x]\n---\nBody\n",
			set:     map[string]any{"status": "done"},
			want:    "---\ntitle: \"a\u2029b\"\nstatus: done\ntags: [x]\n---\nBody\n",
		},
		{
			name:    "next line character inside a quoted value",
			content: "---\ntitle: \"a\u0085b\"\nstatus: draft\ntags: [x]\n---\nBody\n",
			unset:   []string{"status"},
			want:    "---\ntitle: \"a\u0085b\"\ntags: [x]\n---\nBody\n",
		},
		{
			name:    "a lone carriage return separates keys",
			content: "---\ntitle: a\rstatus: draft\n---\nBody\n",
			set:     map[string]any{"status": "done"},
			want:    "---\ntitle: a\rstatus: done\n---\nBody\n",
		},
		{
			name:    "a new key goes before the ... document end marker",
			content: "---\na: 1\n...\nignored: true\n---\nBody\n",
			set:     map[string]any{"b": 2},
			want:    "---\na: 1\nb: 2\n...\nignored: true\n---\nBody\n",
		},
		{
			name:    "an indented comment before a key stays when the key above is replaced",
			content: "---\na: 1\n  # about b\nb: 2\n---\n",
			set:     map[string]any{"a": 5},
			want:    "---\na: 5\n  # about b\nb: 2\n---\n",
		},
		{
			name:    "an indented comment before a key stays when the key above is unset",
			content: "---\na: 1\n  # about b\nb: 2\n---\n",
			unset:   []string{"a"},
			want:    "---\n  # about b\nb: 2\n---\n",
		},
		{
			name:    "a # line inside a block scalar stays with the scalar",
			content: "---\ndesc: |\n  x\n  # not a comment\n---\n",
			set:     map[string]any{"b": 2},
			want:    "---\ndesc: |\n  x\n  # not a comment\nb: 2\n---\n",
		},
		{
			name:    "keep-chomped block scalar keeps its trailing blank lines",
			content: "---\ndesc: |+\n  x\n\n\n---\n",
			set:     map[string]any{"b": 2},
			want:    "---\ndesc: |+\n  x\n\n\nb: 2\n---\n",
		},
		{
			name:    "keep-chomped block scalar lets go of the comment after its blank lines",
			content: "---\ndesc: |+\n  x\n\n# about b\nb: 1\n---\n",
			set:     map[string]any{"desc": "y"},
			want:    "---\ndesc: \"y\"\n# about b\nb: 1\n---\n",
		},
		{
			name:    "an anchor nobody refers to is replaced",
			content: "---\nbase: &b 1\nother: 2\n---\n",
			unset:   []string{"base"},
			want:    "---\nother: 2\n---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := frontmatteredit.Apply(tt.content, tt.set, tt.unset)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			requireNothingLost(t, tt.content, got, tt.set, tt.unset)
		})
	}
}

func TestApplyRefuses(t *testing.T) {
	tests := []struct {
		name    string
		content string
		set     map[string]any
		unset   []string
		wantErr error
	}{
		{
			name:    "nothing to change",
			content: "---\na: 1\n---\n",
			wantErr: frontmatteredit.ErrNothingToChange,
		},
		{
			name:    "empty key",
			content: "---\na: 1\n---\n",
			set:     map[string]any{" ": 1},
			wantErr: frontmatteredit.ErrEmptyKey,
		},
		{
			name:    "key both set and unset",
			content: "---\na: 1\n---\n",
			set:     map[string]any{"a": 2},
			unset:   []string{"a"},
			wantErr: frontmatteredit.ErrSetAndUnset,
		},
		{
			name:    "unclosed frontmatter",
			content: "---\na: 1\nBody\n",
			set:     map[string]any{"a": 2},
			wantErr: frontmatteredit.ErrUnclosed,
		},
		{
			name:    "frontmatter is a list",
			content: "---\n- a\n- b\n---\n",
			set:     map[string]any{"a": 2},
			wantErr: frontmatteredit.ErrNotMapping,
		},
		{
			name:    "flow-style frontmatter",
			content: "---\n{a: 1, b: 2}\n---\n",
			set:     map[string]any{"a": 2},
			wantErr: frontmatteredit.ErrFlowMapping,
		},
		{
			name:    "duplicate key",
			content: "---\na: 1\na: 2\n---\n",
			set:     map[string]any{"a": 3},
			wantErr: frontmatteredit.ErrDuplicateKey,
		},
		{
			name:    "invalid YAML",
			content: "---\na: [1\n---\n",
			set:     map[string]any{"a": 2},
			wantErr: frontmatteredit.ErrInvalidYAML,
		},
		{
			name:    "key with a newline",
			content: "---\na: 1\n---\n",
			set:     map[string]any{"a\nb": 1},
			wantErr: frontmatteredit.ErrControlInKey,
		},
		{
			name:    "unset key with a tab",
			content: "---\na: 1\n---\n",
			unset:   []string{"a\tb"},
			wantErr: frontmatteredit.ErrControlInKey,
		},
		{
			name:    "key with a line separator",
			content: "---\na: 1\n---\n",
			set:     map[string]any{"a\u2028b": 1},
			wantErr: frontmatteredit.ErrControlInKey,
		},
		{
			name:    "indented top-level mapping",
			content: "---\n  a: 1\n  b: 2\n---\nBody\n",
			set:     map[string]any{"c": 3},
			wantErr: frontmatteredit.ErrIndentedMapping,
		},
		{
			name:    "unset an anchored value an alias refers to",
			content: "---\nbase: &b 1\nother: *b\n---\n",
			unset:   []string{"base"},
			wantErr: frontmatteredit.ErrAlias,
		},
		{
			name:    "replace an anchored value an alias refers to",
			content: "---\nbase: &b 1\nother: *b\n---\n",
			set:     map[string]any{"base": 2},
			wantErr: frontmatteredit.ErrAlias,
		},
		{
			name:    "merge key alias",
			content: "---\nbase: &b {x: 1}\nother:\n  <<: *b\n---\n",
			set:     map[string]any{"c": 1},
			wantErr: frontmatteredit.ErrAlias,
		},
		{
			name:    "a value with a fence line would end the frontmatter early",
			content: "---\na: 1\n---\nBody\n",
			set:     map[string]any{"summary": "one\n---\ntwo"},
			wantErr: frontmatteredit.ErrUnsafeEdit,
		},
		{
			name:    "a key goldmark-meta reads under another name",
			content: "---\n~: x\n---\nBody\n",
			set:     map[string]any{"a": 1},
			wantErr: frontmatteredit.ErrUnsafeEdit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := frontmatteredit.Apply(tt.content, tt.set, tt.unset)
			require.ErrorIs(t, err, tt.wantErr)
			require.Empty(t, got)
		})
	}
}
