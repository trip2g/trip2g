package layoutloader

import (
	"reflect"
	"strings"
	"testing"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

// The layouts below are copied verbatim into docs/en/user/templates.md and
// docs/ru/user/jet.md. Change both together.
const (
	docFeaturedLib = `{{ return nvs.ByGlob(. + "/*.md").Public().SortByMeta("order").Limit(3).All() }}`

	docFeaturedCaller = `<ul class="featured">
{{ range _, post := exec("lib/featured", "blog") }}
  <li><a href="{{ post.Permalink() }}">{{ post.Title() }}</a></li>
{{ end }}
</ul>`

	docStatsLib = `{{ notes := nvs.ByGlob(. + "/*.md").Public().All() }}
{{ minutes := 0 }}
{{ range _, n := notes }}
  {{ minutes = minutes + n.ReadingTime() }}
{{ end }}
{{ return map("count", len(notes), "minutes", minutes) }}`

	docStatsCaller = `{{ stats := exec("lib/section_stats", "blog") }}
<p>{{ stats["count"] }} posts, {{ stats["minutes"] }} min of reading</p>`

	docChartWidget = `{{ try }}
  {{ chart := note.M().Get("chart") }}
  <figure class="chart">
    <figcaption>{{ chart["title"] }}</figcaption>
    {{ range _, v := chart["values"] }}
      <span class="bar" style="--v: {{ v }}"></span>
    {{ end }}
  </figure>
{{ catch err }}
  <div class="chart chart--broken">
    Chart unavailable: {{ err.Error() | html }}
  </div>
{{ end }}`

	docRelatedNote = `{{ try }}
  {{ related := nvs.ByPath(note.M().GetString("related", "")) }}
  <aside class="related">
    See also: <a href="{{ related.Permalink() }}">{{ related.Title() }}</a>
  </aside>
{{ end }}`

	docStatsInTry = `{{ try }}
  {{ stats := exec("lib/section_stats", "blog") }}
  <p>{{ stats["count"] }} posts, {{ stats["minutes"] }} min of reading</p>
{{ catch err }}
  {{ if currentUser.IsAdmin() }}
    <p class="admin-error">
      lib/section_stats: {{ err.Error() | html }}
    </p>
  {{ end }}
{{ end }}`
)

func constructsNVS() *templateviews.NVS {
	nvs := model.NewNoteViews()
	nvs.PathMap["blog/first.md"] = &model.NoteView{
		Path: "blog/first.md", Title: "First", Permalink: "/blog/first", Free: true, ReadingTime: 3,
		RawMeta: map[string]interface{}{"order": 2},
	}
	nvs.PathMap["blog/second.md"] = &model.NoteView{
		Path: "blog/second.md", Title: "Second", Permalink: "/blog/second", Free: true, ReadingTime: 5,
		RawMeta: map[string]interface{}{"order": 1},
	}
	nvs.PathMap["blog/paid.md"] = &model.NoteView{
		Path: "blog/paid.md", Title: "Paid", Permalink: "/blog/paid", ReadingTime: 7,
		RawMeta: map[string]interface{}{"order": 0},
	}
	return templateviews.NewNVS(nvs, "live")
}

func constructsVars(page *model.NoteView, admin bool) jet.VarMap {
	vars := make(jet.VarMap)
	vars["nvs"] = reflect.ValueOf(constructsNVS())
	vars["note"] = reflect.ValueOf(templateviews.NewNote(page))
	vars["currentUser"] = reflect.ValueOf(map[string]interface{}{
		"IsAdmin": func() bool { return admin },
	})
	return vars
}

func renderWith(t *testing.T, layouts *model.Layouts, sourceID string, vars jet.VarMap) (string, error) {
	t.Helper()
	layout := layouts.Map[sourceID]
	require.NotNil(t, layout.View, "layout %q has no view: %v", sourceID, layout.Warnings)
	var buf strings.Builder
	err := layout.View.Execute(&buf, vars, nil)
	return buf.String(), err
}

func squash(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func page(id, content string) model.LayoutSourceFile {
	return model.LayoutSourceFile{ID: id, Path: "_layouts" + id + ".html", Content: content}
}

func TestUnderscoreRangeRenders(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ range _, x := items }}[{{ x }}]{{ end }}{{ "a,b" | split(_, ",") | len }}`),
	})
	vars := jet.VarMap{}
	vars.Set("items", []string{"a", "b"})
	out, err := renderWith(t, layouts, "/p", vars)
	require.NoError(t, err)
	require.Equal(t, "[a][b]2", out)
}

func TestTryCatchRendersCatchBranchWithError(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `<p>{{ try }}before{{ missing.Title() }}{{ catch err }}fallback: {{ err.Error() | html }}{{ end }}</p>`),
	})
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "<p>fallback: "), out)
	require.NotContains(t, out, "before", "output of the failed try branch must be discarded")
}

func TestTryWithoutCatchRendersNothingOnFailure(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `<p>{{ try }}x{{ missing.Title() }}{{ end }}</p>`),
	})
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Equal(t, "<p></p>", out)
}

func TestTryCatchYieldsAreAutoImported(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ try }}{{ yield card() }}{{ catch err }}{{ yield fallback() }}{{ end }}`),
		page("/card", `{{ block card() }}CARD{{ end }}`),
		page("/fallback", `{{ block fallback() }}FALLBACK{{ end }}`),
	})
	require.Empty(t, layouts.Map["/p"].Warnings)
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Contains(t, out, "CARD")
}

func TestCatchUsingCurrentUserIsPersonalized(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ try }}ok{{ catch err }}{{ if currentUser.IsAdmin() }}{{ err.Error() }}{{ end }}{{ end }}`),
	})
	require.True(t, layouts.Map["/p"].Personalized)
}

func TestExecReturnsValue(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ x := exec("lib/answer") }}{{ x + 1 }}|{{ exec("lib/answer") }}`),
		page("/lib/answer", `ignored text{{ return 41 }}`),
	})
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Equal(t, "42|41", out)
}

func TestExecReturnedYieldInExecdFileNeedsImport(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ exec("lib/a") }}`),
		page("/lib/a", `{{ import "/card" }}{{ yield card() }}{{ return 1 }}`),
		page("/card", `{{ block card() }}CARD{{ end }}`),
	})
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Equal(t, "1", out, "exec discards output; the import makes the yield resolve")
}

func TestExecPathResolution(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/blog/index", `{{ exec("lib/answer") }}|{{ exec("/lib/answer") }}`),
		page("/lib/answer", `{{ return 42 }}`),
		page("/blog/lib/answer", `{{ return 0 }}`),
	})
	out, err := renderWith(t, layouts, "/blog/index", nil)
	require.NoError(t, err)
	require.Equal(t, "42|42", out, "exec resolves from the _layouts root, not the caller's folder")
}

func TestExecWithHTMLExtensionIsNotFound(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ exec("lib/answer.html") }}`),
		page("/lib/answer", `{{ return 42 }}`),
	})
	_, err := renderWith(t, layouts, "/p", nil)
	require.ErrorContains(t, err, "template /lib/answer.html could not be found")
}

func TestExecMissingFileFailsWithClearError(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ exec("lib/missing") }}`),
	})
	_, err := renderWith(t, layouts, "/p", nil)
	require.ErrorContains(t, err, "template /lib/missing could not be found")
}

func TestDocExecFeatured(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/blog/index", docFeaturedCaller),
		page("/lib/featured", docFeaturedLib),
	})
	out, err := renderWith(t, layouts, "/blog/index", constructsVars(&model.NoteView{}, false))
	require.NoError(t, err)
	require.Equal(t,
		`<ul class="featured"> <li><a href="/blog/second">Second</a></li> <li><a href="/blog/first">First</a></li> </ul>`,
		squash(out))
}

func TestDocExecSectionStats(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/blog/index", docStatsCaller),
		page("/lib/section_stats", docStatsLib),
	})
	out, err := renderWith(t, layouts, "/blog/index", constructsVars(&model.NoteView{}, false))
	require.NoError(t, err)
	require.Equal(t, "<p>2 posts, 8 min of reading</p>", squash(out))
}

func TestDocTryChartWidget(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{page("/note", docChartWidget)})

	good := &model.NoteView{RawMeta: map[string]interface{}{
		"chart": map[string]interface{}{"title": "Visitors", "values": []interface{}{120, 340}},
	}}
	out, err := renderWith(t, layouts, "/note", constructsVars(good, false))
	require.NoError(t, err)
	require.Equal(
		t,
		`<figure class="chart"> <figcaption>Visitors</figcaption> <span class="bar" style="--v: 120"></span> <span class="bar" style="--v: 340"></span> </figure>`,
		squash(out),
	)

	missing := &model.NoteView{RawMeta: map[string]interface{}{}}
	out, err = renderWith(t, layouts, "/note", constructsVars(missing, false))
	require.NoError(t, err)
	require.Contains(t, squash(out), `<div class="chart chart--broken"> Chart unavailable: `)
	require.NotContains(t, out, "<figure")
}

func TestDocTryRelatedNote(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{page("/note", docRelatedNote)})

	linked := &model.NoteView{RawMeta: map[string]interface{}{"related": "blog/first.md"}}
	out, err := renderWith(t, layouts, "/note", constructsVars(linked, false))
	require.NoError(t, err)
	require.Equal(t, `<aside class="related"> See also: <a href="/blog/first">First</a> </aside>`, squash(out))

	dangling := &model.NoteView{RawMeta: map[string]interface{}{"related": "blog/deleted.md"}}
	out, err = renderWith(t, layouts, "/note", constructsVars(dangling, false))
	require.NoError(t, err)
	require.Empty(t, squash(out))
}

func TestDocTryExecStats(t *testing.T) {
	working := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/blog/index", docStatsInTry),
		page("/lib/section_stats", docStatsLib),
	})
	out, err := renderWith(t, working, "/blog/index", constructsVars(&model.NoteView{}, false))
	require.NoError(t, err)
	require.Equal(t, "<p>2 posts, 8 min of reading</p>", squash(out))

	broken := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/blog/index", docStatsInTry),
		page("/lib/section_stats", `{{ return nvs.NoSuchMethod() }}`),
	})
	out, err = renderWith(t, broken, "/blog/index", constructsVars(&model.NoteView{}, false))
	require.NoError(t, err)
	require.Empty(t, squash(out), "visitors see nothing")

	out, err = renderWith(t, broken, "/blog/index", constructsVars(&model.NoteView{}, true))
	require.NoError(t, err)
	require.Contains(t, squash(out), `<p class="admin-error"> lib/section_stats: `)
}

func TestExtendsPageYieldingComponent(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ extends "base" }}{{ import "/extra" }}{{ block main() }}<main>{{ yield card() }}</main>{{ end }}`),
		page("/base", `<html>{{ yield main() }}</html>`),
		page("/extra", `{{ block extra() }}{{ end }}`),
		page("/card", `{{ block card() }}CARD{{ end }}`),
	})
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Equal(t, "<html><main>CARD</main></html>", out)
}

func TestExtendedTemplateYieldNeedsImport(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ extends "base" }}{{ block main() }}M{{ end }}`),
		page("/q", `{{ extends "base_without_import" }}{{ block main() }}M{{ end }}`),
		page("/base", `{{ import "/card" }}<b>{{ yield main() }}{{ yield card() }}</b>`),
		page("/base_without_import", `<b>{{ yield main() }}{{ yield card() }}</b>`),
		page("/card", `{{ block card() }}CARD{{ end }}`),
	})
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Equal(t, "<b>MCARD</b>", out)

	_, err = renderWith(t, layouts, "/q", nil)
	require.ErrorContains(t, err, `unresolved block "card"`)
}

func TestHyphenatedComponentPath(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `<style>{{ yield_blocks("_style_my_theme_") }}</style>{{ yield my_theme_card() }}`),
		page("/my-theme/card", `{{ block _style_@lid() }}.@did{}{{ end }}{{ block @lid() }}<div class="@did">CARD</div>{{ end }}`),
	})
	require.Empty(t, layouts.Map["/my-theme/card"].Warnings)
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Contains(t, out, `<style>.my-theme-card{}</style>`)
	require.Contains(t, out, `<div class="my-theme-card">CARD</div>`)
}

func TestReturnInRenderedPageIsIgnored(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{page("/p", `a{{ return 1 }}b`)})
	out, err := renderWith(t, layouts, "/p", nil)
	require.NoError(t, err)
	require.Equal(t, "ab", out)
}

func TestIncludedTemplateYieldNeedsImport(t *testing.T) {
	layouts := testLoadLayouts(t, []model.LayoutSourceFile{
		page("/p", `{{ include "with_import" }}|{{ include "without_import" }}`),
		page("/with_import", `{{ import "/card" }}{{ yield card() }}`),
		page("/without_import", `{{ yield card() }}`),
		page("/card", `{{ block card() }}CARD{{ end }}`),
	})
	out, err := renderWith(t, layouts, "/p", nil)
	require.ErrorContains(t, err, `unresolved block "card"`)
	require.Equal(t, "CARD|", out)
}
