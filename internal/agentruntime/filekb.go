package agentruntime

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileKB is a KB backed by a directory vault: each document is a file whose
// path is relative to the vault root (forward slashes, matching the scope
// patterns). It is the runnable backend for manual runs; tests use an in-memory
// KB instead.
type FileKB struct {
	root string
}

// NewFileKB builds a FileKB rooted at dir.
func NewFileKB(dir string) *FileKB {
	return &FileKB{root: dir}
}

// resolve converts a vault-relative path into an absolute filesystem path,
// rejecting traversal outside the root.
func (f *FileKB) resolve(path string) (string, error) {
	clean := filepath.Clean("/" + strings.ReplaceAll(path, "\\", "/"))
	abs := filepath.Join(f.root, clean)
	rel, err := filepath.Rel(f.root, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes vault root: %s", path)
	}
	return abs, nil
}

// rel converts an absolute path back to a vault-relative, slash-separated path.
func (f *FileKB) rel(abs string) string {
	rel, err := filepath.Rel(f.root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}

// Search returns documents whose content contains the query (case-insensitive).
func (f *FileKB) Search(_ context.Context, query string) ([]Doc, error) {
	q := strings.ToLower(query)
	var docs []Doc
	walkErr := filepath.WalkDir(f.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path) //nolint:gosec // path is under the vault root.
		if readErr != nil {
			return readErr
		}
		if q == "" || strings.Contains(strings.ToLower(string(data)), q) {
			docs = append(docs, Doc{Path: f.rel(path), Content: snippet(string(data))})
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return docs, nil
}

// Read returns the full content of the document at path.
func (f *FileKB) Read(_ context.Context, path string) (string, error) {
	abs, err := f.resolve(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs) //nolint:gosec // path validated by resolve.
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Write upserts the document at path, creating parent directories as needed.
func (f *FileKB) Write(_ context.Context, path string, content string) error {
	abs, err := f.resolve(path)
	if err != nil {
		return err
	}
	if mkErr := os.MkdirAll(filepath.Dir(abs), 0o755); mkErr != nil {
		return mkErr
	}
	return os.WriteFile(abs, []byte(content), 0o644) //nolint:gosec // vault content is not a secret store.
}

func snippet(s string) string {
	const max = 240
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
