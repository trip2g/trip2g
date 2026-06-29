package rendernotepage_test

import (
	"context"
	"testing"

	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

// renderFreeNoteHTML renders the shared free test note through the real
// default-template page path (Endpoint.Handle -> defaulttemplate.WriteRender) for
// the given viewer token and returns the response HTML body.
func renderFreeNoteHTML(t *testing.T, env *EnvMock, token *usertoken.Data) string {
	t.Helper()
	ctx := newReqCtx(reqOpts{})
	runHandle(t, env, ctx, token)
	return string(body(t, ctx))
}

// TestDefaultTemplate_FreeNoteRenderIsByteStable pins the REAL invariant required
// before the anonymous page cache (PR #54) can be extended to logged-in users:
// the default-template HTML of a free note must be byte-STABLE — the same inputs
// must always produce the same bytes. A page cache stores one rendered body and
// replays it; if the render is nondeterministic, the stored page is an arbitrary
// one of several variants.
//
// RED baseline: this fails today because buildOGTags (endpoint.go) returns a
// map[string]string and the template iterates it with `range ctx.OGTags`
// (views.html), so Go's randomized map order shuffles the <meta property="og:*">
// tags between renders. Rendering the SAME note many times therefore yields
// several distinct byte sequences. Goes GREEN once OG-tag emission is ordered.
func TestDefaultTemplate_FreeNoteRenderIsByteStable(t *testing.T) {
	_, views := cacheTestNote()
	env, _, _ := cacheTestEnv(views, nil)

	first := renderFreeNoteHTML(t, env, nil)
	for i := 1; i < 20; i++ {
		require.Equal(t, first, renderFreeNoteHTML(t, env, nil),
			"default-template render of a free note must be byte-identical across repeated renders (render #%d differs)", i)
	}
}

// TestDefaultTemplate_FreeNoteHTMLIsUserIndependent_NonAdmin is the proof-of-safety
// for extending the page cache to logged-in users: the default-template HTML of a
// free note must be byte-identical for an anonymous viewer and a NON-admin
// logged-in viewer. The live default template has no per-user server branches on
// the note path (the only IsAdmin() branch is the onboarding page), so once OG-tag
// order is deterministic the two renders coincide — meaning the same cached page
// is safe to serve to both. (Admins legitimately differ and stay cache-bypassed.)
//
// This is RED before the OG-order fix (same map-iteration nondeterminism) and
// GREEN after it.
func TestDefaultTemplate_FreeNoteHTMLIsUserIndependent_NonAdmin(t *testing.T) {
	_, views := cacheTestNote()
	env, _, _ := cacheTestEnv(views, nil)
	// Logged-in, non-admin viewer with two active subgraphs. Only this render
	// reaches handleUserToken -> ListActiveUserSubgraphs (resolve.go:440); the live
	// default template never bakes these into the note HTML.
	env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
		return []string{"alpha", "beta"}, nil
	}

	anonHTML := renderFreeNoteHTML(t, env, nil)
	loggedInHTML := renderFreeNoteHTML(t, env, &usertoken.Data{ID: 123, Role: "user"})

	require.Equal(t, anonHTML, loggedInHTML,
		"default-template HTML of a free note must be identical for anon and non-admin logged-in viewers")
}
