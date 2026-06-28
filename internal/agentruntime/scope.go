package agentruntime

import (
	"context"
	"errors"

	"trip2g/internal/webhookutil"
)

// ErrReadDenied is returned when a read targets a path outside read_patterns.
var ErrReadDenied = errors.New("read denied: path outside read scope")

// ErrWriteDenied is returned when a write targets a path outside write_patterns.
var ErrWriteDenied = errors.New("write denied: path outside write scope")

// ScopedKB wraps a KB and enforces read/write glob scopes using the same
// matcher the webhook layer uses (webhookutil.MatchesAny, doublestar globs that
// understand "**"). An empty writePatterns slice denies every write, which is
// exactly how a read-only role (write_patterns: []) is expressed.
type ScopedKB struct {
	kb            KB
	readPatterns  []string
	writePatterns []string
}

// NewScopedKB builds a ScopedKB. The pattern slices are taken as-is (parse them
// from JSON with webhookutil.ParseJSONStringArray at the call site).
func NewScopedKB(kb KB, readPatterns, writePatterns []string) *ScopedKB {
	return &ScopedKB{
		kb:            kb,
		readPatterns:  readPatterns,
		writePatterns: writePatterns,
	}
}

// CanRead reports whether path is within the read scope.
func (s *ScopedKB) CanRead(path string) bool {
	return webhookutil.MatchesAny(path, s.readPatterns)
}

// CanWrite reports whether path is within the write scope.
func (s *ScopedKB) CanWrite(path string) bool {
	return webhookutil.MatchesAny(path, s.writePatterns)
}

// Read returns the document at path, or ErrReadDenied if it is out of scope.
func (s *ScopedKB) Read(ctx context.Context, path string) (string, error) {
	if !s.CanRead(path) {
		return "", ErrReadDenied
	}
	return s.kb.Read(ctx, path)
}

// Search returns only in-scope documents. Out-of-scope hits are dropped so a
// scoped agent can never even see paths outside its read scope.
func (s *ScopedKB) Search(ctx context.Context, query string) ([]Doc, error) {
	docs, err := s.kb.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	filtered := make([]Doc, 0, len(docs))
	for _, d := range docs {
		if s.CanRead(d.Path) {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

// Write upserts the document at path, or returns ErrWriteDenied if out of scope.
func (s *ScopedKB) Write(ctx context.Context, path string, content string) error {
	if !s.CanWrite(path) {
		return ErrWriteDenied
	}
	return s.kb.Write(ctx, path, content)
}
