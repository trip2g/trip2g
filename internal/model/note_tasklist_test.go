package model

import (
	"testing"
)

func TestCountTaskListMarkers(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		want  int
	}{
		{
			name: "no tasks",
			src:  "# Hello\n\nSome text.\n",
			want: 0,
		},
		{
			name: "one unchecked",
			src:  "- [ ] Buy milk\n",
			want: 1,
		},
		{
			name: "one checked",
			src:  "- [x] Done\n",
			want: 1,
		},
		{
			name: "one checked uppercase X",
			src:  "- [X] Done\n",
			want: 1,
		},
		{
			name: "mixed tasks",
			src:  "- [ ] Buy milk\n- [x] Done\n- [ ] Another\n",
			want: 3,
		},
		{
			name: "skips fenced code block with task-like content",
			src:  "```\n- [ ] not a task\n```\n- [ ] real task\n",
			want: 1,
		},
		{
			name: "skips tilde fenced code block",
			src:  "~~~\n- [ ] not a task\n~~~\n- [x] real\n",
			want: 1,
		},
		{
			name: "code block with longer fence",
			src:  "````\n- [ ] inside\n````\n- [ ] outside\n",
			want: 1,
		},
		{
			name: "nested fences not confused",
			src:  "```\n```\n- [ ] still inside outer (no longer fence closes it)\n```\n- [ ] outside\n",
			want: 1,
		},
		{
			name: "task inside text (not list item)",
			src:  "some text [ ] not a task\n",
			want: 0,
		},
		{
			name: "asterisk and plus bullet forms",
			src:  "* [ ] asterisk\n+ [ ] plus\n",
			want: 2,
		},
		{
			name: "indented task list (blockquote nesting won't happen but spaces ok)",
			src:  "  - [ ] indented\n",
			want: 1,
		},
		{
			name: "windows line endings",
			src:  "- [ ] one\r\n- [x] two\r\n",
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountTaskListMarkers([]byte(tt.src))
			if got != tt.want {
				t.Errorf("CountTaskListMarkers(%q) = %d; want %d", tt.src, got, tt.want)
			}
		})
	}
}

func TestTaskListMarkerIndex(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		domCount   int
		n          int
		wantLine   string
		wantLineNo int
	}{
		{
			name:       "first marker",
			src:        "- [ ] Buy milk\n- [x] Done\n",
			domCount:   2,
			n:          0,
			wantLine:   "- [ ] Buy milk",
			wantLineNo: 1,
		},
		{
			name:       "second marker",
			src:        "- [ ] Buy milk\n- [x] Done\n",
			domCount:   2,
			n:          1,
			wantLine:   "- [x] Done",
			wantLineNo: 2,
		},
		{
			name:       "count mismatch returns empty",
			src:        "- [ ] Buy milk\n- [x] Done\n",
			domCount:   3, // wrong DOM count → safety guard fires
			n:          0,
			wantLine:   "",
			wantLineNo: -1,
		},
		{
			name:       "out of range index returns empty",
			src:        "- [ ] one\n",
			domCount:   1,
			n:          5,
			wantLine:   "",
			wantLineNo: -1,
		},
		{
			name:       "skips code fence",
			src:        "```\n- [ ] fake\n```\n- [ ] real\n",
			domCount:   1,
			n:          0,
			wantLine:   "- [ ] real",
			wantLineNo: 4,
		},
		{
			name:       "third of three",
			src:        "- [ ] a\n- [x] b\n- [ ] c\n",
			domCount:   3,
			n:          2,
			wantLine:   "- [ ] c",
			wantLineNo: 3,
		},
		{
			name:       "marker with trailing text",
			src:        "- [ ] Buy milk and eggs\n",
			domCount:   1,
			n:          0,
			wantLine:   "- [ ] Buy milk and eggs",
			wantLineNo: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLine, gotNo := TaskListMarkerIndex([]byte(tt.src), tt.domCount, tt.n)
			if gotLine != tt.wantLine || gotNo != tt.wantLineNo {
				t.Errorf("TaskListMarkerIndex(%q, %d, %d) = (%q, %d); want (%q, %d)",
					tt.src, tt.domCount, tt.n, gotLine, gotNo, tt.wantLine, tt.wantLineNo)
			}
		})
	}
}
