package defaulttemplate

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestJSONLDType(t *testing.T) {
	require.Equal(t, "WebPage", (&Ctx{}).JSONLDType())

	nvs := makeNVS([]*model.NoteView{makeNote("a.md", map[string]interface{}{"schema_type": "HowTo"})})
	require.Equal(t, "HowTo", (&Ctx{Note: nvs.ByPath("a.md")}).JSONLDType())

	nvs = makeNVS([]*model.NoteView{makeNote("p.md", map[string]interface{}{"type": "profile"})})
	require.Equal(t, "ProfilePage", (&Ctx{Note: nvs.ByPath("p.md")}).JSONLDType())

	nvs = makeNVS([]*model.NoteView{makeNote("n.md", nil)})
	require.Equal(t, "BlogPosting", (&Ctx{Note: nvs.ByPath("n.md")}).JSONLDType())
}

func TestShouldEmitJSONLD(t *testing.T) {
	nvs := makeNVS([]*model.NoteView{makeNote("n.md", nil)})
	note := nvs.ByPath("n.md")

	require.False(t, (&Ctx{}).ShouldEmitJSONLD()) // nil note
	require.True(t, (&Ctx{Note: note}).ShouldEmitJSONLD())
	require.False(t, (&Ctx{Note: note, NotFoundMode: true}).ShouldEmitJSONLD())
	require.False(t, (&Ctx{Note: note, OnboardingMode: true}).ShouldEmitJSONLD())
	require.False(t, (&Ctx{Note: note, UnsupportedFileExt: ".canvas"}).ShouldEmitJSONLD())
	require.False(t, (&Ctx{Note: note, MetaRobots: "noindex, nofollow"}).ShouldEmitJSONLD())
	require.False(t, (&Ctx{Note: note, PaywallError: &PaywallError{}}).ShouldEmitJSONLD())
}

func TestJSONLDBreadcrumb(t *testing.T) {
	nvs := makeNVS([]*model.NoteView{makeNote("a/b/c.md", nil)})
	ctx := &Ctx{Note: nvs.ByPath("a/b/c.md"), Notes: nvs, PublicURL: "https://ex.com"}

	crumbs := ctx.JSONLDBreadcrumb()
	require.Len(t, crumbs, 4) // Home, a, b, c
	require.Equal(t, "Home", crumbs[0].Name)
	require.Equal(t, "https://ex.com/", crumbs[0].Item)
	require.Equal(t, "https://ex.com/a/b/c", crumbs[3].Item)

	// Home page (root permalink) → no breadcrumb.
	rootNVS := makeNVS([]*model.NoteView{{Path: "home.md", Permalink: "/"}})
	rootCtx := &Ctx{Note: rootNVS.ByPath("home.md"), Notes: rootNVS, PublicURL: "https://ex.com"}
	require.Nil(t, rootCtx.JSONLDBreadcrumb())
}

func TestDeriveSiteName(t *testing.T) {
	require.Equal(t, "My Blog", DeriveSiteName("%s | My Blog", "https://ex.com"))
	require.Equal(t, "My Blog", DeriveSiteName("%s — My Blog", "https://ex.com"))
	require.Equal(t, "ex.com", DeriveSiteName("%s", "https://ex.com/"))
	require.Equal(t, "ex.com", DeriveSiteName("", "http://ex.com"))
}

func TestJSONLD_RenderValid(t *testing.T) {
	desc := "A short post"
	nv := makeNote("blog/my-post.md", nil)
	nv.Description = &desc
	nv.Lang = "en"
	nv.Author = "Jane Doe"
	nv.Tags = []string{"go", "seo"}
	nv.CreatedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	nvs := makeNVS([]*model.NoteView{nv})

	ctx := &Ctx{
		Note:      nvs.ByPath("blog/my-post.md"),
		Notes:     nvs,
		Title:     "My Post",
		OGTags:    map[string]string{"og:url": "https://ex.com/blog/my-post"},
		PublicURL: "https://ex.com",
		SiteName:  "Example",
	}

	out := JSONLD(ctx)
	inner := extractLDJSON(t, out)

	var doc struct {
		Context string                   `json:"@context"`
		Graph   []map[string]interface{} `json:"@graph"`
	}
	require.NoError(t, json.Unmarshal([]byte(inner), &doc), "JSON-LD must be valid JSON")
	require.Equal(t, "https://schema.org", doc.Context)
	require.GreaterOrEqual(t, len(doc.Graph), 3) // page + breadcrumb + website

	page := doc.Graph[0]
	require.Equal(t, "BlogPosting", page["@type"])
	require.Equal(t, "My Post", page["headline"])
	require.Equal(t, "https://ex.com/blog/my-post", page["url"])
	require.Equal(t, "en", page["inLanguage"])
	require.Equal(t, "A short post", page["description"])
	require.Equal(t, "2026-01-02T00:00:00Z", page["datePublished"])
	require.Equal(t, []interface{}{"go", "seo"}, page["keywords"])
	require.Equal(t, map[string]interface{}{"@type": "Person", "name": "Jane Doe"}, page["author"])
}

func TestJSONLD_EscapesScriptInjection(t *testing.T) {
	nvs := makeNVS([]*model.NoteView{makeNote("n.md", nil)})
	ctx := &Ctx{
		Note:      nvs.ByPath("n.md"),
		Notes:     nvs,
		Title:     "Pwn </script><script>alert(1)</script>",
		OGTags:    map[string]string{"og:url": "https://ex.com/n"},
		PublicURL: "https://ex.com",
		SiteName:  "Example",
	}

	out := JSONLD(ctx)
	// The only literal </script> must be the legitimate closing tag.
	require.Equal(t, 1, strings.Count(out, "</script>"), "injected </script> must be escaped")

	inner := extractLDJSON(t, out)
	var doc struct {
		Graph []map[string]interface{} `json:"@graph"`
	}
	require.NoError(t, json.Unmarshal([]byte(inner), &doc))
	// Round-trips back to the original after JSON decoding.
	require.Equal(t, "Pwn </script><script>alert(1)</script>", doc.Graph[0]["headline"])
}

// extractLDJSON returns the JSON body between the ld+json script tags.
func extractLDJSON(t *testing.T, out string) string {
	t.Helper()
	const open = `<script type="application/ld+json">`
	i := strings.Index(out, open)
	require.GreaterOrEqual(t, i, 0, "missing ld+json script tag")
	rest := out[i+len(open):]
	j := strings.Index(rest, "</script>")
	require.GreaterOrEqual(t, j, 0, "missing closing script tag")
	return rest[:j]
}
