package webhookutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseJSONStringArray(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid JSON array with multiple patterns",
			input:   `["*.md", "docs/*"]`,
			want:    []string{"*.md", "docs/*"},
			wantErr: false,
		},
		{
			name:    "empty JSON array",
			input:   `[]`,
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:        "invalid JSON",
			input:       "not json",
			wantErr:     true,
			errContains: "failed to parse JSON string array",
		},
		{
			name:        "JSON object instead of array",
			input:       `{}`,
			wantErr:     true,
			errContains: "failed to parse JSON string array",
		},
		{
			name:    "single element",
			input:   `["*"]`,
			want:    []string{"*"},
			wantErr: false,
		},
		{
			name:    "doublestar patterns",
			input:   `["**/*.md", "blog/**"]`,
			want:    []string{"**/*.md", "blog/**"},
			wantErr: false,
		},
		{
			name:    "exact match pattern",
			input:   `["readme.md"]`,
			want:    []string{"readme.md"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJSONStringArray(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "matches first pattern",
			path:     "notes/hello.md",
			patterns: []string{"notes/*.md"},
			want:     true,
		},
		{
			name:     "matches second pattern",
			path:     "docs/api.md",
			patterns: []string{"notes/*", "docs/*"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "other/file.txt",
			patterns: []string{"notes/*"},
			want:     false,
		},
		{
			name:     "empty patterns",
			path:     "anything",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "wildcard matches single segment",
			path:     "file.txt",
			patterns: []string{"*"},
			want:     true,
		},
		{
			name:     "wildcard does not match path with slash",
			path:     "any/path",
			patterns: []string{"*"},
			want:     false,
		},
		{
			name:     "doublestar matches nested paths",
			path:     "a/b/c.md",
			patterns: []string{"**/*.md"},
			want:     true,
		},
		{
			name:     "exact match",
			path:     "readme.md",
			patterns: []string{"readme.md"},
			want:     true,
		},
		{
			name:     "exact match does not match subdirectory",
			path:     "docs/readme.md",
			patterns: []string{"readme.md"},
			want:     false,
		},
		{
			name:     "multiple doublestar patterns",
			path:     "blog/2024/post.md",
			patterns: []string{"docs/**", "blog/**"},
			want:     true,
		},
		{
			name:     "pattern with extension",
			path:     "notes/daily/2024-01-01.md",
			patterns: []string{"notes/**/*.md"},
			want:     true,
		},
		{
			name:     "pattern with extension no match",
			path:     "notes/daily/2024-01-01.txt",
			patterns: []string{"notes/**/*.md"},
			want:     false,
		},
		{
			name:     "invalid pattern is skipped",
			path:     "test.txt",
			patterns: []string{"[invalid", "*.txt"},
			want:     true,
		},
		{
			name:     "all invalid patterns return false",
			path:     "test.txt",
			patterns: []string{"[invalid", "[another"},
			want:     false,
		},
		{
			name:     "empty path matches wildcard",
			path:     "",
			patterns: []string{"*"},
			want:     true,
		},
		{
			name:     "root level file with doublestar",
			path:     "file.md",
			patterns: []string{"**/*.md"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesAny(tt.path, tt.patterns)
			require.Equal(t, tt.want, got)
		})
	}
}
