package agentruntime

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
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

	// allowRoleAuthoring disables the role-authoring guard (see roleguard.go).
	// The zero value keeps the guard on, so every construction is safe by
	// default and only an explicit operator opt-in turns it off.
	allowRoleAuthoring bool
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
// Backslashes are treated as separators (like FileKB.resolve) so
// "concepts/..\secrets/x.md" is seen as traversal here too — the authz and
// resolution layers must agree on what a path means.
func normalizeScopePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
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
	norm, err := s.checkWrite(path, content)
	if err != nil {
		return err
	}
	return s.kb.Write(ctx, norm, content)
}

// checkWrite is Write's pre-flight — scope and the role guard — without the
// write. It returns the normalized path the KB would receive.
func (s *ScopedKB) checkWrite(path, content string) (string, error) {
	norm, err := s.checkWriteScope(path)
	if err != nil {
		return "", err
	}
	if !s.allowRoleAuthoring && declaresRole(content) {
		return "", ErrRoleAuthoringDenied
	}
	return norm, nil
}

// checkWriteScope normalizes path and denies it when outside write_patterns.
func (s *ScopedKB) checkWriteScope(path string) (string, error) {
	norm := normalizeScopePath(path)
	if norm == "" || !webhookutil.MatchesAny(norm, s.writePatterns) {
		return "", ErrWriteDenied
	}
	return norm, nil
}

// Patch applies a find/replace to the document at path, or returns
// ErrWriteDenied if out of write scope. The normalized path is forwarded to
// the KB, same as Write.
func (s *ScopedKB) Patch(ctx context.Context, path, find, replace string) error {
	norm, err := s.checkWriteScope(path)
	if err != nil {
		return err
	}
	verified, err := s.checkPatchNotRoleAuthoring(ctx, norm, find, replace)
	if err != nil {
		return err
	}
	return s.applyPatch(ctx, norm, find, replace, verified)
}

// applyPatch is Patch's second half: the edit itself, conditional on the
// verified bytes when there are any and the KB can honour the condition.
func (s *ScopedKB) applyPatch(ctx context.Context, norm, find, replace string, verified *string) error {
	if cp, ok := s.kb.(conditionalPatcher); ok && verified != nil {
		return cp.PatchIfUnchanged(ctx, norm, find, replace, contentHash(*verified))
	}
	return s.kb.Patch(ctx, norm, find, replace)
}

// conditionalPatcher is the optional half of KB: a backend that can apply a
// patch only if the note still hashes to what the caller verified. remoteKB
// implements it by passing expectedHash to trip2g, which compares against the
// live content inside the same atomic mutation. A KB without it (FileKB, test
// doubles) is patched unconditionally, as before.
type conditionalPatcher interface {
	PatchIfUnchanged(ctx context.Context, path, find, replace, expectedHash string) error
}

// contentHash mirrors trip2g's hashContent (internal/case/updatenotes) byte for
// byte — base64 URL encoding of the SHA-256 of the raw content. The two are
// pinned to the same golden value from both sides; see TestContentHashGolden
// here and its twin in the updatenotes tests. A silent drift between them would
// turn every conditional patch into a hash mismatch.
func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return base64.URLEncoding.EncodeToString(sum[:])
}

// checkPatchNotRoleAuthoring denies a patch that would leave a role note behind,
// and one that edits a note already carrying the marker.
//
// A patch is a find/replace applied server-side (trip2g's updateNotes), so
// fleet does not otherwise read the note: verifying costs one read. Matching on
// the replace fragment alone would be cheaper and is not enough — retagging an
// existing role note changes only the fleet_id VALUE, so the marker never
// appears in the fragment.
//
// The read goes through the underlying KB, not this ScopedKB, on purpose: a
// role may hold write scope over a path without read scope, and the guard must
// not be defeated by the role's own read_patterns.
// It returns the exact bytes it verified, so the caller can make the mutation
// conditional on them and close the window between checking and patching. nil
// means nothing was verified (the guard is off), not that the note was empty.
func (s *ScopedKB) checkPatchNotRoleAuthoring(ctx context.Context, path, find, replace string) (*string, error) {
	if s.allowRoleAuthoring {
		return nil, nil
	}
	current, err := s.readForPatch(ctx, path)
	if err != nil {
		return nil, err
	}
	return s.guardPatch(current, find, replace)
}

// readForPatch reads the note a patch targets, through the underlying KB (see
// checkPatchNotRoleAuthoring). With the guard on, a failed read is
// ErrRoleGuardUnverifiable: the guard cannot run, so it fails closed.
func (s *ScopedKB) readForPatch(ctx context.Context, path string) (string, error) {
	current, err := s.kb.Read(ctx, path)
	if err != nil && !s.allowRoleAuthoring {
		return "", fmt.Errorf("%w: %w", ErrRoleGuardUnverifiable, err)
	}
	return current, err
}

// guardPatch is the role guard's decision on a patch to a note whose current
// content is already in hand. It returns the bytes it verified, for the caller
// to condition the patch on; nil when the guard is off.
func (s *ScopedKB) guardPatch(current, find, replace string) (*string, error) {
	if s.allowRoleAuthoring {
		return nil, nil
	}
	patched, _ := applyPatchPreview(current, find, replace)
	if declaresRole(current) || declaresRole(patched) {
		return nil, ErrRoleAuthoringDenied
	}
	return &current, nil
}
