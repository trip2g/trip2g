package rendernotepage

import (
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func makeTestNoteViews() *model.NoteViews {
	return model.NewNoteViews()
}

func makeTestNote(permalink, path string, routes []model.ParsedRoute) *model.NoteView {
	note := &model.NoteView{
		Permalink:         permalink,
		PermalinkOriginal: permalink,
		Path:              path,
		Free:              true,
		Routes:            routes,
	}
	return note
}

func TestResolveNote_MainDomain_RouteAlias(t *testing.T) {
	nvs := makeTestNoteViews()
	note := makeTestNote("/about-note", "about-note.md", []model.ParsedRoute{
		{Host: "", Path: "/about"},
	})
	nvs.RegisterNote(note)

	result := resolveNote(nvs, "example.com", "/about", "https://example.com")
	require.Equal(t, note, result)
}

func TestResolveNote_MainDomain_NoCollision(t *testing.T) {
	// _index.md is served at "/" via nv.Map; another note has route: /
	// resolveNote on main domain checks RouteMap[""] first, so routeNote wins for /
	nvs := makeTestNoteViews()

	indexNote := makeTestNote("/", "_index.md", nil)
	indexNote.IsIndex = true
	nvs.RegisterNote(indexNote)

	routeNote := makeTestNote("/landing", "landing.md", []model.ParsedRoute{
		{Host: "", Path: "/"},
	})
	nvs.RegisterNote(routeNote)

	result := resolveNote(nvs, "example.com", "/", "https://example.com")
	require.Equal(t, routeNote, result)
}

func TestResolveNote_CustomDomain_Root(t *testing.T) {
	nvs := makeTestNoteViews()
	note := makeTestNote("/landing", "landing.md", []model.ParsedRoute{
		{Host: "foo.com", Path: "/"},
	})
	nvs.RegisterNote(note)

	result := resolveNote(nvs, "foo.com", "/", "https://example.com")
	require.Equal(t, note, result)
}

func TestResolveNote_CustomDomain_SubPage(t *testing.T) {
	nvs := makeTestNoteViews()
	note := makeTestNote("/hello-note", "hello.md", []model.ParsedRoute{
		{Host: "foo.com", Path: "/hello"},
	})
	nvs.RegisterNote(note)

	result := resolveNote(nvs, "foo.com", "/hello", "https://example.com")
	require.Equal(t, note, result)
}

func TestResolveNote_CustomDomain_Fallthrough(t *testing.T) {
	// Unknown path on custom domain falls back to nv.Map
	nvs := makeTestNoteViews()
	note := makeTestNote("/my-page", "my-page.md", nil)
	nvs.RegisterNote(note)

	result := resolveNote(nvs, "foo.com", "/my-page", "https://example.com")
	require.Equal(t, note, result)
}

func TestResolveNote_MainDomain_Unchanged(t *testing.T) {
	nvs := makeTestNoteViews()
	note := makeTestNote("/existing-page", "existing-page.md", nil)
	nvs.RegisterNote(note)

	result := resolveNote(nvs, "example.com", "/existing-page", "https://example.com")
	require.Equal(t, note, result)
}

func TestResolveNote_MainDomainRoute_NotOnCustomDomain(t *testing.T) {
	// A main domain alias route: /x is NOT served on foo.com/x
	nvs := makeTestNoteViews()
	note := makeTestNote("/note", "note.md", []model.ParsedRoute{
		{Host: "", Path: "/x"}, // main domain alias only
	})
	nvs.RegisterNote(note)

	// On custom domain, /x is not found via RouteMap, and /x is not in nv.Map
	result := resolveNote(nvs, "foo.com", "/x", "https://example.com")
	require.Nil(t, result)
}

func TestResolveNote_NotFound(t *testing.T) {
	nvs := makeTestNoteViews()
	result := resolveNote(nvs, "example.com", "/nonexistent", "https://example.com")
	require.Nil(t, result)
}
