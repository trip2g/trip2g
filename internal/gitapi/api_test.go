package gitapi

import (
	"reflect"
	"testing"
)

func TestFilterDotFiles(t *testing.T) {
	api := &API{}

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "no dot files",
			input:    []string{"README.md", "docs/guide.md", "src/main.go"},
			expected: []string{"README.md", "docs/guide.md", "src/main.go"},
		},
		{
			name:     "dot files at root",
			input:    []string{".gitignore", ".env", "README.md"},
			expected: []string{"README.md"},
		},
		{
			name:     "dot files in subdirectories",
			input:    []string{"docs/.hidden", "src/main.go", "test/.cache/file.txt"},
			expected: []string{"src/main.go"},
		},
		{
			name:     "mixed files with dots in names",
			input:    []string{"config.yml", ".secret", "file.name.ext", "dir/.hidden/file.md"},
			expected: []string{"config.yml", "file.name.ext"},
		},
		{
			name:     "empty strings filtered out",
			input:    []string{"", "file.md", "", ".hidden"},
			expected: []string{"file.md"},
		},
		{
			name:     "russian filenames with dots",
			input:    []string{"русский.md", ".скрытый", "папка/.конфиг", "обычный/файл.txt"},
			expected: []string{"русский.md", "обычный/файл.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.filterDotFiles(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterDotFiles() = %v, want %v", result, tt.expected)
			}
		})
	}
}
