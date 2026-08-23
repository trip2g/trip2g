package rendernotepage_test

import (
	"net/http"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func loadTestVault(t *testing.T, sources []mdloader.SourceFile) *model.NoteViews {
	t.Helper()
	log := logger.TestLogger{}
	views, err := mdloader.Load(mdloader.Options{
		Sources:   sources,
		Log:       &log,
		Version:   "live",
		PublicURL: "https://example.com",
	})
	require.NoError(t, err)
	return views
}

// A note with an explicit route must answer 200 at the route path itself,
// independent of what characters its vault filename happens to contain.
// Today the alternate-permalink 301 (endpoint.go) fires for any request path
// that differs from Permalink once AlternatePermalinks is non-nil, so two
// notes in the same folder with identical route frontmatter behave
// differently: the ascii one answers 200, the transliterated one redirects.
func TestRouteAlias_ServedDirectly_RegardlessOfFilename(t *testing.T) {
	views := loadTestVault(t, []mdloader.SourceFile{
		{
			Path:    "docs/note-a.md",
			Content: []byte("---\nfree: true\nroute: /short-a\n---\nAscii filename note"),
		},
		{
			Path:    "docs/статья.md",
			Content: []byte("---\nfree: true\nroute: /short-b\n---\nTransliterated filename note"),
		},
	})

	env, _, _ := cacheTestEnv(views, nil)

	tests := []struct {
		name string
		path string
	}{
		{name: "ascii filename", path: "/short-a"},
		{name: "transliterated filename", path: "/short-b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newReqCtx(reqOpts{path: tt.path})
			runHandle(t, env, ctx, nil)
			require.Equal(t, http.StatusOK, ctx.Response.StatusCode(),
				"route path %s must be served directly, got redirect to %q",
				tt.path, string(ctx.Response.Header.Peek("Location")))
		})
	}
}

// Same inconsistency on a custom domain, where it is worse: the 301 points at
// the note's vault-path permalink, but the custom domain serves only explicit
// routes, so the redirect target answers 404 on that host.
func TestRouteOnCustomDomain_ServedAtRoutePath(t *testing.T) {
	views := loadTestVault(t, []mdloader.SourceFile{
		{
			Path:    "docs/статья.md",
			Content: []byte("---\nfree: true\nroute: docs.example.com/c\n---\nRouted note"),
		},
	})

	env, _, _ := cacheTestEnv(views, nil)

	ctx := newReqCtx(reqOpts{host: "docs.example.com", path: "/c"})
	runHandle(t, env, ctx, nil)

	status := ctx.Response.StatusCode()
	location := string(ctx.Response.Header.Peek("Location"))

	locationRoutable := location == "" || views.GetByRoute("docs.example.com", location) != nil

	require.Equal(t, http.StatusOK, status,
		"route path must be served directly on its domain; got %d to %q (routable on this host: %v)",
		status, location, locationRoutable)
}
