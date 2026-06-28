package rendernotepage

import (
	"encoding/json"
	"html"
	"strconv"
	"strings"

	"trip2g/internal/case/renderlayout"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/langdetect"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/valyala/fasthttp"
)

// userSpaceHelper builds the window.__trip2g_settings script block and the
// user-space JS bundle <script> tags for custom Jet layouts.
//
// Idempotent: the settings JSON block is emitted only on the FIRST call to
// scripts(); any subsequent call emits only the <script src defer> tags.
// This lets a layout safely call {{ default_template.user_space_scripts() }}
// once in <head> without duplicating the inline settings object.
type userSpaceHelper struct {
	jsURLs          []string
	cssURLs         []string
	localeHashes    map[string]string
	uiLang          string
	devMode         bool
	isAdmin         bool
	title           string
	note            *templateviews.Note
	nvs             *templateviews.NVS
	layoutSections  []model.LayoutSectionEntry
	settingsEmitted bool
}

// newUserSpaceHelper constructs the helper from pre-extracted data values.
// Nil slices/maps are normalised to empty to avoid "null" in JSON output.
func newUserSpaceHelper(
	jsURLs []string,
	localeHashes map[string]string,
	uiLang string,
	devMode bool,
	isAdmin bool,
	title string,
	note *templateviews.Note,
) *userSpaceHelper {
	if jsURLs == nil {
		jsURLs = []string{}
	}
	if localeHashes == nil {
		localeHashes = map[string]string{}
	}
	return &userSpaceHelper{
		jsURLs:       jsURLs,
		localeHashes: localeHashes,
		uiLang:       uiLang,
		devMode:      devMode,
		isAdmin:      isAdmin,
		title:        title,
		note:         note,
	}
}

// scripts returns the raw HTML string to inject.
//
// Jet calls this via reflection when the template uses
// {{ default_template.user_space_scripts() }}.  Because the Jet set used by
// custom layouts is created with jet.WithSafeWriter(nil), the returned string
// is written directly to the response without HTML-escaping.
func (h *userSpaceHelper) scripts() string {
	var sb strings.Builder

	if !h.settingsEmitted {
		h.settingsEmitted = true

		devMode := "false"
		if h.devMode {
			devMode = "true"
		}

		// Use the standard library JSON encoder.  json.Marshal escapes <, >, &
		// as \uXXXX sequences — this is safe inside <script> blocks and prevents
		// accidental </script> injection in edge-case titles.
		jsURLsJSON, _ := json.Marshal(h.jsURLs)
		localeHashesJSON, _ := json.Marshal(h.localeHashes)
		titleJSON, _ := json.Marshal(h.title)
		uiLangJSON, _ := json.Marshal(h.uiLang)

		sb.WriteString("<script>\nwindow.__trip2g_settings = {\n")
		sb.WriteString("  is_dev_mode: ")
		sb.WriteString(devMode)
		sb.WriteString(",\n")
		sb.WriteString("  title: ")
		sb.Write(titleJSON)
		sb.WriteString(",\n")
		sb.WriteString("  ui_lang: ")
		sb.Write(uiLangJSON)
		sb.WriteString(",\n")
		sb.WriteString("  js_urls: ")
		sb.Write(jsURLsJSON)
		sb.WriteString(",\n")
		sb.WriteString("  locale_hashes: ")
		sb.Write(localeHashesJSON)

		if h.note != nil {
			noteLangJSON, _ := json.Marshal(h.note.Lang())
			notePathJSON, _ := json.Marshal(h.note.Path())
			noteVersionJSON, _ := json.Marshal(h.note.VersionID())

			sb.WriteString(",\n")
			sb.WriteString("  note_lang: ")
			sb.Write(noteLangJSON)
			sb.WriteString(",\n")
			sb.WriteString("  note_path: ")
			sb.Write(notePathJSON)
			sb.WriteString(",\n")
			sb.WriteString("  note_path_id: ")
			sb.WriteString(strconv.FormatInt(h.note.PathID(), 10))
			sb.WriteString(",\n")
			sb.WriteString("  note_version_id: ")
			sb.Write(noteVersionJSON)
		}

		sb.WriteString("\n}\n</script>\n")
	}

	for _, u := range h.jsURLs {
		sb.WriteString(`<script src="`)
		sb.WriteString(html.EscapeString(u))
		sb.WriteString(`" defer></script>`)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// admin returns true when the current viewer is the site owner/admin.
// Jet calls this via reflection for {{ currentUser.IsAdmin() }}.
func (h *userSpaceHelper) admin() bool {
	return h.isAdmin
}

// dtCtx builds the minimal default-template Ctx needed to resolve and render
// the site header/footer chrome for a custom layout.
func (h *userSpaceHelper) dtCtx() *defaulttemplate.Ctx {
	return &defaulttemplate.Ctx{
		Note:           h.note,
		Notes:          h.nvs,
		LayoutSections: h.layoutSections,
	}
}

// header returns the standard site-header HTML (or "" when the page has no
// header). Jet calls this for {{ defaultTemplate.Header() }}.
func (h *userSpaceHelper) header() string {
	return defaulttemplate.Header(h.dtCtx())
}

// footer returns the standard site-footer HTML (or "" when the page has no
// footer). Jet calls this for {{ defaultTemplate.Footer() }}.
func (h *userSpaceHelper) footer() string {
	return defaulttemplate.Footer(h.dtCtx())
}

// styles returns <link> tags for the default-template stylesheet so a custom
// layout embedding the standard chrome isn't unstyled. Reuses the existing
// hashed served URL (renderlayout.Env.UserCSSURLs) — never a hardcoded hash.
// Jet calls this for {{ defaultTemplate.Styles() }}.
func (h *userSpaceHelper) styles() string {
	var sb strings.Builder
	for _, u := range h.cssURLs {
		sb.WriteString(`<link rel="stylesheet" href="`)
		sb.WriteString(html.EscapeString(u))
		sb.WriteString(`">`)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// jetMap returns the map stored in vars["defaultTemplate"] (and its deprecated
// vars["default_template"] alias). Holds template-chrome helpers: script
// injection plus the standard header/footer/styles.
func (h *userSpaceHelper) jetMap() map[string]interface{} {
	return map[string]interface{}{
		"UserSpaceScripts":   h.scripts,
		"user_space_scripts": h.scripts, // deprecated alias
		"Header":             h.header,
		"Footer":             h.footer,
		"Styles":             h.styles,
	}
}

// currentUserJetMap returns the map stored in vars["currentUser"] (and its
// deprecated vars["current_user"] alias). Layouts call IsAdmin() to gate edit
// affordances.
func (h *userSpaceHelper) currentUserJetMap() map[string]interface{} {
	return map[string]interface{}{
		"IsAdmin":  h.admin,
		"is_admin": h.admin, // deprecated alias
	}
}

// buildUserSpaceHelper constructs the helper for renderLayout.
// It sources data the same way buildDefaultTemplateCtx does:
//   - jsURLs and localeHashes from renderlayout.Env
//   - uiLang via langdetect using cookie + Accept-Language header
//   - devMode from renderlayout.Env
//   - title and note from the resolve response
//   - isAdmin from resp.UserToken.IsAdmin() (nil-safe)
func buildUserSpaceHelper(
	ctx *fasthttp.RequestCtx,
	env Env,
	resp *Response,
) *userSpaceHelper {
	var (
		jsURLs       []string
		cssURLs      []string
		localeHashes map[string]string
		devMode      bool
	)

	if rlEnv, ok := env.(renderlayout.Env); ok {
		jsURLs = rlEnv.UserJSURLs()
		cssURLs = rlEnv.UserCSSURLs()
		localeHashes = rlEnv.UserLocaleHashes()
		devMode = rlEnv.IsDevMode()
	}

	uiLang := langdetect.DetectPreferred(
		string(ctx.Request.Header.Cookie(langCookieName)),
		string(ctx.Request.Header.Peek("Accept-Language")),
	)

	h := newUserSpaceHelper(jsURLs, localeHashes, uiLang, devMode, resp.UserToken.IsAdmin(), resp.Title, resp.NoteView)
	h.cssURLs = cssURLs
	h.nvs = templateviews.NewNVS(resp.Notes, resp.DefaultVersion)
	if resp.Notes != nil {
		h.layoutSections = resp.Notes.LayoutSections
	}
	return h
}
