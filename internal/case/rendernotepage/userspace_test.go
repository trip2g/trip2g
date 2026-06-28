package rendernotepage

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

// sampleNote builds a minimal *templateviews.Note for tests.
func sampleNote() *templateviews.Note {
	nv := &model.NoteView{
		Path:      "notes/hello.md",
		PathID:    42,
		VersionID: 99,
		Lang:      "en",
	}
	return templateviews.NewNote(nv)
}

// TestUserSpaceHelper_SettingsFields verifies that the rendered output contains
// the __trip2g_settings JSON with all required fields.
func TestUserSpaceHelper_SettingsFields(t *testing.T) {
	h := newUserSpaceHelper(
		[]string{"/assets/bundle.js"},
		map[string]string{"en": "abc123", "ru": "def456"},
		"en",
		false,
		false,
		"My Title",
		sampleNote(),
	)

	out := h.scripts()

	// Settings block must appear.
	require.Contains(t, out, "window.__trip2g_settings")

	// Scalar fields.
	require.Contains(t, out, "is_dev_mode: false")
	require.Contains(t, out, `"My Title"`)
	require.Contains(t, out, `"en"`)

	// js_urls as a JSON array (unquoted JS key, JSON array value).
	require.Contains(t, out, `js_urls: [`)
	require.Contains(t, out, `"/assets/bundle.js"`)

	// locale_hashes as a JSON object (unquoted JS key, JSON object value).
	require.Contains(t, out, `locale_hashes: {`)
	require.Contains(t, out, `"abc123"`)

	// Note fields (unquoted JS keys, JSON-quoted values).
	require.Contains(t, out, "note_lang: ")
	require.Contains(t, out, "note_path: ")
	require.Contains(t, out, "note_path_id: 42")
	require.Contains(t, out, "note_version_id: ")

	// Bundle script tag.
	require.Contains(t, out, `<script src="/assets/bundle.js" defer></script>`)
}

// TestUserSpaceHelper_DevModeTrue verifies is_dev_mode renders "true" when set.
func TestUserSpaceHelper_DevModeTrue(t *testing.T) {
	h := newUserSpaceHelper(nil, nil, "ru", true, false, "Title", nil)
	out := h.scripts()
	require.Contains(t, out, "is_dev_mode: true")
}

// TestUserSpaceHelper_Idempotent verifies that calling scripts() twice emits
// the __trip2g_settings block exactly once.
func TestUserSpaceHelper_Idempotent(t *testing.T) {
	h := newUserSpaceHelper(
		[]string{"/assets/bundle.js"},
		nil, "en", false, false, "Title", nil,
	)

	first := h.scripts()
	second := h.scripts()

	// First call: settings block + script tag.
	require.Contains(t, first, "window.__trip2g_settings")
	require.Contains(t, first, `<script src="/assets/bundle.js" defer></script>`)

	// Second call: only script tags, no duplicate settings block.
	require.NotContains(t, second, "window.__trip2g_settings")
	require.Contains(t, second, `<script src="/assets/bundle.js" defer></script>`)

	// Combined output: settings appears exactly once.
	combined := first + second
	count := strings.Count(combined, "window.__trip2g_settings")
	require.Equal(t, 1, count, "settings block must appear exactly once across two calls")
}

// TestUserSpaceHelper_NoNoteFields verifies that note_lang/note_path etc. are
// absent when the helper is built without a note.
func TestUserSpaceHelper_NoNoteFields(t *testing.T) {
	h := newUserSpaceHelper(nil, nil, "en", false, false, "Title", nil)
	out := h.scripts()
	require.NotContains(t, out, "note_lang")
	require.NotContains(t, out, "note_path")
	require.NotContains(t, out, "note_path_id")
	require.NotContains(t, out, "note_version_id")
}

// TestUserSpaceHelper_RawOutput verifies that the returned HTML is not
// entity-escaped (i.e., <script> tags are literal, not &lt;script&gt;).
func TestUserSpaceHelper_RawOutput(t *testing.T) {
	h := newUserSpaceHelper(
		[]string{"/assets/bundle.js"},
		nil, "en", false, false, "Title", nil,
	)
	out := h.scripts()
	require.NotContains(t, out, "&lt;")
	require.NotContains(t, out, "&gt;")
	require.Contains(t, out, "<script")
	require.Contains(t, out, "</script>")
}

// testStringLoader is a minimal jet.Loader backed by an in-memory map.
type testStringLoader struct {
	templates map[string]string
}

func (l *testStringLoader) Open(path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(l.templates[path])), nil
}

func (l *testStringLoader) Exists(path string) bool {
	_, ok := l.templates[path]
	return ok
}

// TestJetBinding_UserSpaceScripts verifies that the dotted Jet call
// {{ defaultTemplate.UserSpaceScripts() }} resolves and renders correctly,
// and that the output is NOT HTML-escaped (WithSafeWriter(nil) behaviour).
func TestJetBinding_UserSpaceScripts(t *testing.T) {
	const tmplKey = "/test.html"
	loader := &testStringLoader{
		templates: map[string]string{
			tmplKey: `{{ defaultTemplate.UserSpaceScripts() }}`,
		},
	}

	// Replicate the exact Jet set options used by layoutloader (no escaping).
	views := jet.NewSet(loader, jet.DevelopmentMode(true), jet.WithSafeWriter(nil))
	tmpl, err := views.GetTemplate(tmplKey)
	require.NoError(t, err)

	h := newUserSpaceHelper(
		[]string{"/assets/app.js", "/assets/extra.js"},
		map[string]string{"en": "hash1"},
		"ru",
		true,
		false,
		"Jet Test",
		sampleNote(),
	)

	vars := make(jet.VarMap)
	vars["defaultTemplate"] = reflect.ValueOf(h.jetMap())

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars, nil)
	require.NoError(t, err)

	out := buf.String()

	// Settings block present.
	require.Contains(t, out, "window.__trip2g_settings")
	require.Contains(t, out, "is_dev_mode: true")
	require.Contains(t, out, `"Jet Test"`)
	require.Contains(t, out, `"ru"`)
	require.Contains(t, out, `js_urls: [`)
	require.Contains(t, out, `"/assets/app.js"`)
	require.Contains(t, out, `locale_hashes: {`)
	require.Contains(t, out, `"hash1"`)

	// Note fields present.
	require.Contains(t, out, "note_path_id: 42")

	// Script src tags present (both bundles).
	require.Contains(t, out, `<script src="/assets/app.js" defer></script>`)
	require.Contains(t, out, `<script src="/assets/extra.js" defer></script>`)

	// Not entity-escaped.
	require.NotContains(t, out, "&lt;script")
	require.NotContains(t, out, "&lt;/script")
}

// loaderTestEnv satisfies layoutloader.Env for testing.
type loaderTestEnv struct{ log logger.Logger }

func (e *loaderTestEnv) Logger() logger.Logger { return e.log }

// TestJetBinding_IdempotentViaLayoutloader verifies idempotency through the
// full layoutloader pipeline (covers the real jet.WithSafeWriter(nil) path).
func TestJetBinding_IdempotentViaLayoutloader(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:   "/two-calls",
		Path: "_layouts/two-calls.html",
		// Call UserSpaceScripts twice in the same render.
		Content: `{{ defaultTemplate.UserSpaceScripts() }}MIDDLE{{ defaultTemplate.UserSpaceScripts() }}`,
	}}

	env := &loaderTestEnv{log: &logger.TestLogger{}}

	layouts, err := layoutloader.Load(env, sources, layoutloader.Options{})
	require.NoError(t, err)

	h := newUserSpaceHelper(
		[]string{"/assets/bundle.js"},
		nil, "en", false, false, "Title", nil,
	)

	vars := make(jet.VarMap)
	vars["defaultTemplate"] = reflect.ValueOf(h.jetMap())

	var buf bytes.Buffer
	err = layouts.Map["/two-calls"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	out := buf.String()

	// settings block appears exactly once.
	count := strings.Count(out, "window.__trip2g_settings")
	require.Equal(t, 1, count, "settings block must appear exactly once when called twice in template")

	// Script tags appear twice (once per call).
	scriptCount := strings.Count(out, `<script src="/assets/bundle.js" defer></script>`)
	require.Equal(t, 2, scriptCount, "bundle script tag should appear once per call")
}

// TestJetBinding_IsAdmin_True verifies {{ currentUser.IsAdmin() }} renders
// "true" (Go bool true printed by Jet's fastprinter) when isAdmin=true.
func TestJetBinding_IsAdmin_True(t *testing.T) {
	const tmplKey = "/admin-check.html"
	loader := &testStringLoader{
		templates: map[string]string{
			tmplKey: `{{ currentUser.IsAdmin() }}`,
		},
	}
	views := jet.NewSet(loader, jet.DevelopmentMode(true), jet.WithSafeWriter(nil))
	tmpl, err := views.GetTemplate(tmplKey)
	require.NoError(t, err)

	h := newUserSpaceHelper(nil, nil, "en", false, true, "Title", nil)
	vars := make(jet.VarMap)
	vars["currentUser"] = reflect.ValueOf(h.currentUserJetMap())

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars, nil)
	require.NoError(t, err)
	require.Equal(t, "true", buf.String())
}

// TestJetBinding_IsAdmin_False verifies {{ currentUser.IsAdmin() }} renders
// "false" for a non-admin viewer.
func TestJetBinding_IsAdmin_False(t *testing.T) {
	const tmplKey = "/admin-check-false.html"
	loader := &testStringLoader{
		templates: map[string]string{
			tmplKey: `{{ currentUser.IsAdmin() }}`,
		},
	}
	views := jet.NewSet(loader, jet.DevelopmentMode(true), jet.WithSafeWriter(nil))
	tmpl, err := views.GetTemplate(tmplKey)
	require.NoError(t, err)

	h := newUserSpaceHelper(nil, nil, "en", false, false, "Title", nil)
	vars := make(jet.VarMap)
	vars["currentUser"] = reflect.ValueOf(h.currentUserJetMap())

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars, nil)
	require.NoError(t, err)
	require.Equal(t, "false", buf.String())
}

// stubPartialRenderer is a no-op model.NoteViewPartialRenderer so SiteFooter can
// render its chrome wrapper without a full markdown pipeline in tests.
type stubPartialRenderer struct{}

func (stubPartialRenderer) Sections(int) []model.NoteViewSection      { return nil }
func (stubPartialRenderer) Section(string) *model.NoteViewSection     { return nil }
func (stubPartialRenderer) Introduce() model.NoteViewSection          { return model.NoteViewSection{} }
func (stubPartialRenderer) HeadingBlocks(int) []model.NoteViewSection { return nil }
func (stubPartialRenderer) FirstList() *model.NoteViewList            { return nil }
func (stubPartialRenderer) Lists() []model.NoteViewList               { return nil }
func (stubPartialRenderer) FirstImageURL() string                    { return "" }

// chromeHelper builds a userSpaceHelper whose Notes resolve _header/_footer so
// that Header()/Footer()/Styles() produce the standard chrome HTML.
func chromeHelper() *userSpaceHelper {
	mainNV := &model.NoteView{Path: "blog/post1.md", Permalink: "/blog/post1"}
	headerNV := &model.NoteView{Path: "_header.md", Permalink: "/_header"}
	footerNV := &model.NoteView{Path: "_footer.md", Permalink: "/_footer", PartialRenderer: stubPartialRenderer{}}

	nvs := model.NewNoteViews()
	for _, nv := range []*model.NoteView{mainNV, headerNV, footerNV} {
		nvs.PathMap[nv.Path] = nv
		nvs.Map[nv.Permalink] = nv
	}
	tv := templateviews.NewNVS(nvs, "live")

	h := newUserSpaceHelper(nil, nil, "en", false, false, "Title", tv.ByPath("blog/post1.md"))
	h.nvs = tv
	h.cssURLs = []string{"/assets/defaulttemplate.css?h=abc123"}
	return h
}

// TestJetBinding_DefaultTemplateChrome_CamelCase verifies the new camelCase
// namespace + PascalCase methods resolve and render raw (WithSafeWriter(nil)):
// UserSpaceScripts(), Header(), Footer(), Styles().
func TestJetBinding_DefaultTemplateChrome_CamelCase(t *testing.T) {
	const tmplKey = "/chrome.html"
	loader := &testStringLoader{
		templates: map[string]string{
			tmplKey: `[s]{{ defaultTemplate.UserSpaceScripts() }}[h]{{ defaultTemplate.Header() }}` +
				`[f]{{ defaultTemplate.Footer() }}[c]{{ defaultTemplate.Styles() }}`,
		},
	}
	views := jet.NewSet(loader, jet.DevelopmentMode(true), jet.WithSafeWriter(nil))
	tmpl, err := views.GetTemplate(tmplKey)
	require.NoError(t, err)

	h := chromeHelper()
	vars := make(jet.VarMap)
	vars["defaultTemplate"] = reflect.ValueOf(h.jetMap())

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars, nil)
	require.NoError(t, err)
	out := buf.String()

	require.Contains(t, out, "window.__trip2g_settings")    // UserSpaceScripts()
	require.Contains(t, out, `<header class="site-header">`) // Header()
	require.Contains(t, out, `<footer class="site-footer">`) // Footer()
	require.Contains(t, out, `<link rel="stylesheet" href="/assets/defaulttemplate.css?h=abc123">`) // Styles()

	// Chrome HTML is written raw, not entity-escaped.
	require.NotContains(t, out, "&lt;header")
	require.NotContains(t, out, "&lt;footer")
	require.NotContains(t, out, "&lt;link")
}

// TestJetBinding_IsAdmin_CamelCase verifies {{ currentUser.IsAdmin() }} resolves.
func TestJetBinding_IsAdmin_CamelCase(t *testing.T) {
	const tmplKey = "/current-user.html"
	loader := &testStringLoader{
		templates: map[string]string{
			tmplKey: `{{ currentUser.IsAdmin() }}`,
		},
	}
	views := jet.NewSet(loader, jet.DevelopmentMode(true), jet.WithSafeWriter(nil))
	tmpl, err := views.GetTemplate(tmplKey)
	require.NoError(t, err)

	h := newUserSpaceHelper(nil, nil, "en", false, true, "Title", nil)
	vars := make(jet.VarMap)
	vars["currentUser"] = reflect.ValueOf(h.currentUserJetMap())

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars, nil)
	require.NoError(t, err)
	require.Equal(t, "true", buf.String())
}

