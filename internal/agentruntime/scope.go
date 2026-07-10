package agentruntime

import (
	"context"
	"errors"
	pathpkg "path"
	"strings"

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

// NewScopedKB builds a ScopedKB. Patterns are normalized (leading "/" and "./"
// stripped) so a config like "/concepts/**" matches the slash-less candidates
// produced by normalizeScopePath instead of silently denying everything.
func NewScopedKB(kb KB, readPatterns, writePatterns []string) *ScopedKB {
	return &ScopedKB{
		kb:            kb,
		readPatterns:  normalizeScopePatterns(readPatterns),
		writePatterns: normalizeScopePatterns(writePatterns),
	}
}

// stripScopePrefix removes leading "./" and "/" segments. Repeat because
// "/./"-style prefixes can stack.
func stripScopePrefix(p string) string {
	for {
		switch {
		case strings.HasPrefix(p, "./"):
			p = p[2:]
		case strings.HasPrefix(p, "/"):
			p = p[1:]
		default:
			return p
		}
	}
}

// normalizeScopePatterns strips leading "/" and "./" from each pattern so
// patterns compare against normalized (slash-less) candidate paths. Patterns
// are not path.Clean-ed to keep glob segments like "**" intact.
func normalizeScopePatterns(patterns []string) []string {
	if patterns == nil {
		return nil
	}
	out := make([]string, len(patterns))
	for i, p := range patterns {
		out[i] = stripScopePrefix(strings.TrimSpace(p))
	}
	return out
}

// normalizeScopePath cleans a candidate path to a scope-relative form before
// glob-matching. Small LLM models sometimes prepend "/" or "./" to a path, so
// "/concepts/x.md" must match "concepts/**". It strips leading "./" and "/",
// then path.Clean-resolves any "." / ".." segments. If the cleaned path still
// escapes the scope root (leading "..", ".", or empty), it returns "" — a
// sentinel that never matches any pattern, so traversal ("../x",
// "concepts/../../etc/passwd") and absolute-escape stay denied.
func normalizeScopePath(p string) string {
	p = stripScopePrefix(strings.TrimSpace(p))
	if p == "" {
		return ""
	}
	c := pathpkg.Clean(p)
	// After cleaning, a leading ".." (or bare "." / "..") means the path resolved
	// outside the scope root: deny by returning the never-match sentinel.
	if c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return ""
	}
	return c
}

// CanRead reports whether path is within the read scope.
func (s *ScopedKB) CanRead(path string) bool {
	norm := normalizeScopePath(path)
	if norm == "" {
		return false
	}
	return webhookutil.MatchesAny(norm, s.readPatterns)
}

// CanWrite reports whether path is within the write scope.
func (s *ScopedKB) CanWrite(path string) bool {
	norm := normalizeScopePath(path)
	if norm == "" {
		return false
	}
	return webhookutil.MatchesAny(norm, s.writePatterns)
}

// Read returns the document at path, or ErrReadDenied if it is out of scope.
// The normalized path is what reaches the underlying KB, so slash-prefixed
// inputs resolve to the same document as their canonical form.
func (s *ScopedKB) Read(ctx context.Context, path string) (string, error) {
	norm := normalizeScopePath(path)
	if norm == "" || !webhookutil.MatchesAny(norm, s.readPatterns) {
		return "", ErrReadDenied
	}
	return s.kb.Read(ctx, norm)
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
// The normalized path is forwarded to the KB so a slash-prefixed path can't
// create a duplicate ghost document.
func (s *ScopedKB) Write(ctx context.Context, path string, content string) error {
	norm := normalizeScopePath(path)
	if norm == "" || !webhookutil.MatchesAny(norm, s.writePatterns) {
		return ErrWriteDenied
	}
	return s.kb.Write(ctx, norm, content)
}

// Patch applies a find/replace to the document at path, or returns
// ErrWriteDenied if out of write scope. The normalized path is forwarded to
// the KB, same as Write.
func (s *ScopedKB) Patch(ctx context.Context, path, find, replace string) error {
	norm := normalizeScopePath(path)
	if norm == "" || !webhookutil.MatchesAny(norm, s.writePatterns) {
		return ErrWriteDenied
	}
	return s.kb.Patch(ctx, norm, find, replace)
}
