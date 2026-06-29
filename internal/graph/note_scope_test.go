package graph

// Tests for read_pattern scope enforcement helpers:
//
//   filterNotePathsByScope   — guards NotePaths resolver (ByValues/Like/AllVisible branches)
//   resolveFsPathFromPermalink — guards Note resolver fsPath lookup (scope uses filesystem
//                                path, not permalink/URL)
//   wikilinkTargetAllowed    — guards ResolveWikilinks resolver (G3 regression)
//
// F1 regression: notePaths ByValues/Like/AllVisible branches had ZERO read_pattern
// enforcement. A scoped shortapitoken could list any filesystem path via those
// branches. The fix wraps each branch in filterNotePathsByScope.
//
// F1 regression (Note resolver): the read-pattern check was matching against the
// URL permalink instead of the filesystem path. The fix uses NoteView.Path (the
// filesystem path) so that read & write patterns share one namespace.
//
// G3 regression: resolveWikilinks returned target.Path+URL with no read_patterns
// check — leaks existence/paths of out-of-scope notes to a scoped token.
// The fix calls wikilinkTargetAllowed before exposing any target fields.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	appmodel "trip2g/internal/model"
)

// scopedCtx returns a plain context with an appreq stamped as a scoped
// shortapitoken request with the given read_patterns. Used to test helpers
// that call appreq.Scoped(ctx) and appreq.WebhookReadPatterns(ctx).
func scopedCtx(readPatterns []string) context.Context {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetRequestURI("http://example.com/graphql")
	req := &appreq.Request{
		Req:                 fctx,
		WebhookScoped:       true,
		WebhookReadPatterns: readPatterns,
	}
	req.StoreInContext()
	return fctx
}

// unscopedCtx returns a plain context with an appreq that is NOT scoped
// (simulating admin / personal-token / api-key auth). Used to test that
// unscoped requests bypass read_pattern filtering.
func unscopedCtx() context.Context {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetRequestURI("http://example.com/graphql")
	req := &appreq.Request{
		Req:           fctx,
		WebhookScoped: false,
	}
	req.StoreInContext()
	return fctx
}

// -- filterNotePathsByScope --

func TestFilterNotePathsByScope_ScopedToken_FiltersOutOfScope(t *testing.T) {
	// A scoped token with read_patterns:["boards/**"] must hide paths outside
	// that glob. This is the core NotePaths F1 regression.
	paths := []db.NotePath{
		{ID: 1, Value: "boards/sprint.md"},
		{ID: 2, Value: "docs/readme.md"},    // out of scope
		{ID: 3, Value: "boards/retro.md"},
		{ID: 4, Value: "private/secret.md"}, // out of scope
	}

	ctx := scopedCtx([]string{"boards/**"})

	got := filterNotePathsByScope(ctx, paths)

	require.Len(t, got, 2)
	require.Equal(t, "boards/sprint.md", got[0].Value)
	require.Equal(t, "boards/retro.md", got[1].Value)
}

func TestFilterNotePathsByScope_ScopedToken_AllInScope(t *testing.T) {
	paths := []db.NotePath{
		{ID: 1, Value: "boards/sprint.md"},
		{ID: 2, Value: "boards/retro.md"},
	}

	ctx := scopedCtx([]string{"boards/**"})
	got := filterNotePathsByScope(ctx, paths)
	require.Equal(t, paths, got)
}

func TestFilterNotePathsByScope_ScopedToken_EmptyInput(t *testing.T) {
	ctx := scopedCtx([]string{"boards/**"})
	got := filterNotePathsByScope(ctx, nil)
	require.Empty(t, got)
}

func TestFilterNotePathsByScope_UnscopedRequest_PassthroughAll(t *testing.T) {
	// Admin / personal-token requests are not scoped — all paths pass through.
	paths := []db.NotePath{
		{ID: 1, Value: "boards/sprint.md"},
		{ID: 2, Value: "private/secret.md"},
	}

	ctx := unscopedCtx()
	got := filterNotePathsByScope(ctx, paths)
	require.Equal(t, paths, got, "unscoped request must not filter any paths")
}

// TestFilterNotePathsByScope_ScopedEmptyReadPatterns_DeniesAll is the G1
// regression test. Before the fix, a scoped token with empty read_patterns was
// fail-open (len(rp)==0 → early return → all paths visible). The fix drives
// enforcement off appreq.Scoped() so empty patterns deny all reads.
func TestFilterNotePathsByScope_ScopedEmptyReadPatterns_DeniesAll(t *testing.T) {
	paths := []db.NotePath{
		{ID: 1, Value: "boards/sprint.md"},
		{ID: 2, Value: "docs/readme.md"},
	}

	// Scoped token with explicitly empty read_patterns — no read access granted.
	ctx := scopedCtx([]string{})

	got := filterNotePathsByScope(ctx, paths)
	require.Nil(t, got, "scoped token with empty read_patterns must deny all reads (fail-closed)")
}

func TestFilterNotePathsByScope_MultiplePatterns(t *testing.T) {
	paths := []db.NotePath{
		{ID: 1, Value: "boards/sprint.md"},
		{ID: 2, Value: "docs/guide.md"},
		{ID: 3, Value: "private/secret.md"}, // out of scope
	}

	ctx := scopedCtx([]string{"boards/**", "docs/**"})
	got := filterNotePathsByScope(ctx, paths)
	require.Len(t, got, 2)
	values := []string{got[0].Value, got[1].Value}
	require.Contains(t, values, "boards/sprint.md")
	require.Contains(t, values, "docs/guide.md")
}

// -- resolveFsPathFromPermalink --

// TestResolveFsPathFromPermalink_UsesFsPathNotPermalink is the key regression
// for the Note resolver. Before the fix, the scope check used nv.Permalink
// (the URL path) instead of nv.Path (the filesystem path). A note with
// Permalink="/boards/sprint" and Path="boards/sprint.md" would fail a
// "boards/**" read-pattern match against the permalink.
//
// After the fix, resolveFsPathFromPermalink returns nv.Path so that the scope
// check uses the filesystem namespace — the same as write_patterns.
func TestResolveFsPathFromPermalink_UsesFsPathNotPermalink(t *testing.T) {
	// Simulate a note whose URL and filesystem paths differ in extension.
	// Before the fix: matching "/boards/sprint" against "boards/**" → false (miss)
	// After the fix: matching "boards/sprint.md" against "boards/**" → true (hit)
	views := &appmodel.NoteViews{
		Map: map[string]*appmodel.NoteView{
			"/boards/sprint": {
				Path:      "boards/sprint.md",   // filesystem path
				Permalink: "/boards/sprint",     // URL path
			},
		},
	}

	got := resolveFsPathFromPermalink(views, "/boards/sprint")
	require.Equal(t, "boards/sprint.md", got,
		"must return filesystem Path, not Permalink, for read_pattern matching")
}

func TestResolveFsPathFromPermalink_MissingView_ReturnsEmpty(t *testing.T) {
	views := &appmodel.NoteViews{
		Map: map[string]*appmodel.NoteView{},
	}
	got := resolveFsPathFromPermalink(views, "/boards/sprint")
	require.Equal(t, "", got)
}

func TestResolveFsPathFromPermalink_NilViews_ReturnsEmpty(t *testing.T) {
	got := resolveFsPathFromPermalink(nil, "/boards/sprint")
	require.Equal(t, "", got)
}

func TestResolveFsPathFromPermalink_EmptyPermalink_ReturnsEmpty(t *testing.T) {
	views := &appmodel.NoteViews{
		Map: map[string]*appmodel.NoteView{
			"/boards/sprint": {Path: "boards/sprint.md"},
		},
	}
	got := resolveFsPathFromPermalink(views, "")
	require.Equal(t, "", got)
}

// -- wikilinkTargetAllowed (G3 regression) --

// TestWikilinkTargetAllowed_ScopedToken_OutOfScope_Denied is the G3 regression
// test. Before the fix, resolveWikilinks set res.Path/res.URL for any resolved
// target regardless of the token's read_patterns — leaking out-of-scope note
// existence/paths to a scoped agent. The fix gates each target on
// wikilinkTargetAllowed before exposing Path or URL.
func TestWikilinkTargetAllowed_ScopedToken_OutOfScope_Denied(t *testing.T) {
	ctx := scopedCtx([]string{"boards/**"})
	// "docs/guide.md" is outside the scoped token's read_patterns.
	allowed := wikilinkTargetAllowed(ctx, "docs/guide.md")
	require.False(t, allowed, "scoped token must not see targets outside read_patterns")
}

func TestWikilinkTargetAllowed_ScopedToken_InScope_Allowed(t *testing.T) {
	ctx := scopedCtx([]string{"boards/**"})
	allowed := wikilinkTargetAllowed(ctx, "boards/sprint.md")
	require.True(t, allowed, "scoped token must see targets inside read_patterns")
}

func TestWikilinkTargetAllowed_ScopedToken_EmptyPatterns_DeniesAll(t *testing.T) {
	// Scoped token with empty read_patterns is fail-closed: no targets allowed.
	ctx := scopedCtx([]string{})
	allowed := wikilinkTargetAllowed(ctx, "boards/sprint.md")
	require.False(t, allowed, "scoped token with empty read_patterns must deny all targets (fail-closed)")
}

func TestWikilinkTargetAllowed_UnscopedToken_AlwaysAllowed(t *testing.T) {
	// Admin / personal-token requests are not scoped — all targets pass through.
	ctx := unscopedCtx()
	allowed := wikilinkTargetAllowed(ctx, "private/secret.md")
	require.True(t, allowed, "unscoped request must not filter any wikilink targets")
}

func TestWikilinkTargetAllowed_ScopedToken_MultiplePatterns(t *testing.T) {
	ctx := scopedCtx([]string{"boards/**", "docs/**"})

	cases := []struct {
		path    string
		allowed bool
	}{
		{"boards/sprint.md", true},
		{"docs/guide.md", true},
		{"private/secret.md", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			got := wikilinkTargetAllowed(ctx, tc.path)
			require.Equal(t, tc.allowed, got)
		})
	}
}

