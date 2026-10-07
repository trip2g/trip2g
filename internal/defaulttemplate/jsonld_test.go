package defaulttemplate

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

func TestJSONLDType(t *testing.T) {
	require.Equal(t, "WebPage", (&Ctx{}).JSONLDType())

	nvs := makeNVS([]*model.NoteView{makeNote("a.md", map[string]interface{}{"schema_type": "HowTo"})})
	require.Equal(t, "HowTo", (&Ctx{Note: nvs.NoteByPath("a.md")}).JSONLDType())

	nvs = makeNVS([]*model.NoteView{makeNote("p.md", map[string]interface{}{"type": "profile"})})
	require.Equal(t, "ProfilePage", (&Ctx{Note: nvs.NoteByPath("p.md")}).JSONLDType())

	nvs = makeNVS([]*model.NoteView{makeNote("n.md", nil)})
	require.Equal(t, "BlogPosting", (&Ctx{Note: nvs.NoteByPath("n.md")}).JSONLDType())
}

func TestShouldEmitJSONLD(t *testing.T) {
	nvs := makeNVS([]*model.NoteView{makeNote("n.md", nil)})
	note := nvs.NoteByPath("n.md")

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
	ctx := &Ctx{Note: nvs.NoteByPath("a/b/c.md"), Notes: nvs, PublicURL: "https://ex.com"}

	crumbs := ctx.JSONLDBreadcrumb()
	require.Len(t, crumbs, 4) // Home, a, b, c
	require.Equal(t, "Home", crumbs[0].Name)
	require.Equal(t, "https://ex.com/", crumbs[0].Item)
	require.Equal(t, "https://ex.com/a/b/c", crumbs[3].Item)

	// Home page (root permalink) → no breadcrumb.
	rootNVS := makeNVS([]*model.NoteView{{Path: "home.md", Permalink: "/"}})
	rootCtx := &Ctx{Note: rootNVS.NoteByPath("home.md"), Notes: rootNVS, PublicURL: "https://ex.com"}
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
		Note:      nvs.NoteByPath("blog/my-post.md"),
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
		Note:      nvs.NoteByPath("n.md"),
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

// TestJSONLDImage_IsAbsolute pins the fix for broken schema.org image/logo
// fields: note asset URLs are site-relative (see model.NoteAssetURLPath), but
// JSON-LD is consumed by external crawlers (Google rich results, social
// unfurlers), so image/logo must be absolute — same as og:image.
func TestJSONLDImage_IsAbsolute(t *testing.T) {
	nv := makeNote("blog/my-post.md", map[string]interface{}{"og_image": "cover.png"})
	nv.AssetReplaces = map[string]*model.NoteAssetReplace{
		"cover.png": {Hash: "abc", FileName: "cover.png", URL: model.NoteAssetURLPath("abc", "cover.png")},
	}
	nvs := makeNVS([]*model.NoteView{nv})

	ctx := &Ctx{Note: nvs.NoteByPath("blog/my-post.md"), PublicURL: "https://ex.com"}

	require.Equal(t, "https://ex.com/_system/assets/abc/cover.png", ctx.JSONLDImage())
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

func TestJSONLDBreadcrumbFromVisibleCrumbs(t *testing.T) {
	page := func(path, frontmatter string) mdloader.SourceFile {
		return mdloader.SourceFile{Path: path, Content: []byte("---\n" + frontmatter + "\n---\nbody")}
	}
	srcs := []mdloader.SourceFile{
		{Path: "docs/_sidebar.md", Content: []byte("### Start\n\n- [[a]]\n\n### [[guide|Guide]]\n\n- [[b]]\n")},
		page("docs/a.md", "title: A\nleft_sidebar: docs/_sidebar.md"),
		page("docs/b.md", "title: B\nleft_sidebar: docs/_sidebar.md"),
		page("docs/guide.md", "title: Guide page"),
		page("docs/fm.md", `title: FM
breadcrumbs:
  - {label: Home, href: /}
  - "[[docs/guide]]"
  - {label: Plain}
  - {label: Ext, href: "https://other.com/x"}`),
		page("docs/sub/none.md", "title: None"),
	}
	for i := range srcs {
		srcs[i].PathID = int64(i + 1)
	}
	pages, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &logger.TestLogger{}})
	require.NoError(t, err)
	nvs := templateviews.NewNVS(pages, "")

	home := JSONLDCrumb{Name: "Home", Item: "https://ex.com/"}
	tests := []struct {
		name string
		path string
		want []JSONLDCrumb
	}{
		{
			name: "plain-text crumb is skipped",
			path: "docs/a.md",
			want: []JSONLDCrumb{home, {Name: "A", Item: "https://ex.com/docs/a"}},
		},
		{
			name: "linked heading crumb",
			path: "docs/b.md",
			want: []JSONLDCrumb{
				home,
				{Name: "Guide", Item: "https://ex.com/docs/guide"},
				{Name: "B", Item: "https://ex.com/docs/b"},
			},
		},
		{
			name: "frontmatter crumbs, home not repeated",
			path: "docs/fm.md",
			want: []JSONLDCrumb{
				home,
				{Name: "Guide page", Item: "https://ex.com/docs/guide"},
				{Name: "Ext", Item: "https://other.com/x"},
				{Name: "FM", Item: "https://ex.com/docs/fm"},
			},
		},
		{
			name: "no visible crumbs falls back to the URL path",
			path: "docs/sub/none.md",
			want: []JSONLDCrumb{
				home,
				{Name: "docs", Item: "https://ex.com/docs"},
				{Name: "sub", Item: "https://ex.com/docs/sub"},
				{Name: "None", Item: "https://ex.com/docs/sub/none"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{Note: nvs.NoteByPath(tt.path), Notes: nvs, PublicURL: "https://ex.com"}
			require.Equal(t, tt.want, ctx.JSONLDBreadcrumb())

			ctx.OGTags = map[string]string{"og:url": "https://ex.com" + ctx.Note.Permalink()}
			var doc struct {
				Graph []struct {
					Type  string `json:"@type"`
					Items []struct {
						Position int    `json:"position"`
						Item     string `json:"item"`
					} `json:"itemListElement"`
				} `json:"@graph"`
			}
			require.NoError(t, json.Unmarshal([]byte(extractLDJSON(t, JSONLD(ctx))), &doc))
			require.Equal(t, "BreadcrumbList", doc.Graph[1].Type)
			for i, item := range doc.Graph[1].Items {
				require.Equal(t, i+1, item.Position)
				require.NotEmpty(t, item.Item)
			}
		})
	}
}
