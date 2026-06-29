package layoutloader

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

// countingLoader counts Open calls. In Jet, each Open corresponds to one
// template read (io.ReadAll) followed by a parse — exactly the work that
// DevelopmentMode=true repeats on every render.
type countingLoader struct {
	templates map[string]string
	opens     map[string]int
}

func (l *countingLoader) Exists(p string) bool {
	_, ok := l.templates[p]
	return ok
}

func (l *countingLoader) Open(p string) (io.ReadCloser, error) {
	l.opens[p]++
	return io.NopCloser(strings.NewReader(l.templates[p])), nil
}

// TestDevMode_CachesParsedIncludeAcrossRenders is the core perf assertion.
//
// `{{ include }}` is resolved at RUNTIME (jet eval.go executeInclude ->
// getSiblingTemplate). With DevelopmentMode(false) the parsed include is cached
// in the Set after the first render and reused; with DevelopmentMode(true) the
// Set skips its cache and re-reads + re-parses the include on EVERY render.
//
// The jet.NewSet construction here mirrors loader.go's NewSet call exactly.
func TestDevMode_CachesParsedIncludeAcrossRenders(t *testing.T) {
	templates := map[string]string{
		"/main": `<main>{{ include "/part" }}</main>`,
		"/part": `<p>shared</p>`,
	}

	renderNTimes := func(devMode bool, n int) int {
		loader := &countingLoader{templates: templates, opens: map[string]int{}}
		set := jet.NewSet(loader, jet.DevelopmentMode(devMode), jet.WithSafeWriter(nil))
		tmpl, err := set.GetTemplate("/main")
		require.NoError(t, err)
		for range n {
			var buf bytes.Buffer
			require.NoError(t, tmpl.Execute(&buf, nil, nil))
			require.Equal(t, "<main><p>shared</p></main>", buf.String())
		}
		return loader.opens["/part"]
	}

	// Production behavior: the include is parsed once, then served from cache.
	require.Equal(t, 1, renderNTimes(false, 5),
		"DevMode=false must parse the included template exactly once across renders")

	// Dev behavior (live reload): re-read + re-parse on every render.
	require.Equal(t, 5, renderNTimes(true, 5),
		"DevMode=true re-parses the included template on every render")
}

// TestLoadDevModeFalse_LayoutEditsPropagateAcrossReloads is the correctness
// guard: turning caching ON (DevMode=false) must NOT pin stale layouts.
//
// layoutloader.Load builds a fresh jet.Set per reload from the CURRENT
// sourceFiles, so an edited layout (a new Load) renders the new content even
// though DevelopmentMode is off — the cache lives only for one Set's lifetime.
func TestLoadDevModeFalse_LayoutEditsPropagateAcrossReloads(t *testing.T) {
	// devMode=false: production caching ON (IsDevMode returns false).
	env := &testEnv{logger: &logger.TestLogger{}, devMode: false}
	mk := func(body string) []model.LayoutSourceFile {
		return []model.LayoutSourceFile{{
			ID:      "/page",
			Path:    "_layouts/page.html",
			Content: body,
		}}
	}

	render := func(sources []model.LayoutSourceFile) string {
		layouts, err := Load(env, sources, Options{})
		require.NoError(t, err)
		var buf bytes.Buffer
		require.NoError(t, layouts.Map["/page"].View.Execute(&buf, nil, nil))
		return buf.String()
	}

	require.Equal(t, "<h1>v1</h1>", render(mk(`<h1>v1</h1>`)))
	// Edited layout pushed -> new Load -> new Set: edit must take effect.
	require.Equal(t, "<h1>v2-edited</h1>", render(mk(`<h1>v2-edited</h1>`)),
		"layout edits must propagate across reloads even with DevMode=false")
}

// TestLoadDevMode_OutputIdenticalAcrossModes proves the DevMode flag changes
// only parse caching, never rendered output, through the public Load API with a
// runtime {{ include }} (the path that re-parses in dev mode).
func TestLoadDevMode_OutputIdenticalAcrossModes(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/page", Path: "_layouts/page.html", Content: `<main>{{ include "/part" }}</main>`},
		{ID: "/part", Path: "_layouts/part.html", Content: `<p>shared</p>`},
	}

	out := func(dev bool) string {
		// env.IsDevMode() drives jet.DevelopmentMode via the Env interface.
		env := &testEnv{logger: &logger.TestLogger{}, devMode: dev}
		layouts, err := Load(env, sources, Options{})
		require.NoError(t, err)
		// Render twice to exercise the include cache/no-cache path.
		require.NoError(t, layouts.Map["/page"].View.Execute(io.Discard, nil, nil))
		var buf bytes.Buffer
		require.NoError(t, layouts.Map["/page"].View.Execute(&buf, nil, nil))
		return buf.String()
	}

	require.Equal(t, out(true), out(false), "DevMode must not change rendered output")
	require.Contains(t, out(false), "<main><p>shared</p></main>")
}
