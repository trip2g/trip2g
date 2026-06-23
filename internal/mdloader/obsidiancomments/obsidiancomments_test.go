package obsidiancomments_test

import (
	"bytes"
	"testing"
	"trip2g/internal/mdloader/obsidiancomments"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

func newMD() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(obsidiancomments.ObsidianComments),
	)
}

func convert(t *testing.T, src string) string {
	t.Helper()
	var buf bytes.Buffer
	err := newMD().Convert([]byte(src), &buf)
	require.NoError(t, err)
	return buf.String()
}

func TestObsidianComments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "inline comment stripped",
			in:   "This is an %%invisible%% comment.",
			want: "<p>This is an  comment.</p>\n",
		},
		{
			name: "inline comment at start of line",
			in:   "%%hidden%% visible",
			want: "<p> visible</p>\n",
		},
		{
			name: "inline comment at end of line",
			in:   "visible %%hidden%%",
			want: "<p>visible </p>\n",
		},
		{
			name: "block comment produces no output",
			in:   "before\n%%\nthis is\na block comment\n%%\nafter",
			want: "<p>before</p>\n<p>after</p>\n",
		},
		{
			name: "multiple inline comments in one line",
			in:   "a %%b%% c %%d%% e",
			want: "<p>a  c  e</p>\n",
		},
		{
			name: "multiple comments in one document",
			in:   "%%first%% text %%second%%\n\n%%\nblock\n%%\n\nend",
			want: "<p> text </p>\n<p>end</p>\n",
		},
		{
			name: "unmatched lone %% left alone",
			in:   "text %% more text",
			want: "<p>text %% more text</p>\n",
		},
		{
			name: "percent percent inside fenced code block untouched",
			in:   "```\n%%comment%%\n```",
			want: "<pre><code>%%comment%%\n</code></pre>\n",
		},
		{
			name: "percent percent inside inline code untouched",
			in:   "`%%not a comment%%`",
			want: "<p><code>%%not a comment%%</code></p>\n",
		},
		{
			name: "empty comment",
			in:   "text %%%% more",
			want: "<p>text  more</p>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convert(t, tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}
