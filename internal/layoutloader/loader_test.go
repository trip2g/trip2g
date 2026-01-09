package layoutloader

import (
	"bytes"
	"reflect"
	"testing"
	"time"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	logger logger.Logger
}

func (t *testEnv) Logger() logger.Logger {
	return t.logger
}

func TestResolveAssets(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/trip2g/main",
		VersionID: 27,
		Path:      "_layouts/trip2g/main.html",
		Content:   `{{ asset("style.css") }}, {{ asset("main.js") }}`,
		Assets: map[string]*model.NoteAssetReplace{
			"_layouts/trip2g/style.css": &model.NoteAssetReplace{
				URL:  "https://storage/style.css",
				Hash: "abc123",
			},
			"_layouts/trip2g/main.js": &model.NoteAssetReplace{
				URL:  "https://storage/main.js",
				Hash: "def456",
			},
		},
	}}

	options := Options{}
	env := &testEnv{logger: &logger.TestLogger{}}

	layouts, err := Load(env, sources, options)
	require.NoError(t, err)
	require.Len(t, layouts.Map, 1)

	var buf bytes.Buffer

	err = layouts.Map["/trip2g/main"].View.Execute(&buf, nil, nil)
	require.NoError(t, err)

	require.Len(t, layouts.Map["/trip2g/main"].Assets, 2)
	require.Equal(t, "_layouts/trip2g/style.css", layouts.Map["/trip2g/main"].Assets[0].Path)
	require.Equal(t, "_layouts/trip2g/main.js", layouts.Map["/trip2g/main"].Assets[1].Path)

	require.Equal(t, `https://storage/style.css, https://storage/main.js`, buf.String())
}

func TestYieldBlocks(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/trip2g/main",
		VersionID: 27,
		Path:      "_layouts/trip2g/main.html",
		Content:   `{{ import "blocks" }}{{ yield main_layout() content }}hello{{ end }}`,
	}, {
		ID:        "/trip2g/blocks",
		VersionID: 28,
		Path:      "_layouts/trip2g/blocks.html",
		Content:   `{{ block main_layout() }}<wrapper>{{ yield content }}</wrapper>{{ end }}`,
	}}

	options := Options{}
	env := &testEnv{logger: &logger.TestLogger{}}

	layouts, err := Load(env, sources, options)
	require.NoError(t, err)
	require.Len(t, layouts.Map, 2)
}

// createTestNVS creates NVS with test notes for template tests.
func createTestNVS() *templateviews.NVS {
	nvs := model.NewNoteViews()

	nvs.PathMap["blog/hello-world.md"] = &model.NoteView{
		Path:      "blog/hello-world.md",
		Title:     "Hello World",
		Permalink: "/blog/hello-world",
		CreatedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		RawMeta: map[string]interface{}{
			"order": 2,
		},
	}
	nvs.PathMap["blog/getting-started.md"] = &model.NoteView{
		Path:      "blog/getting-started.md",
		Title:     "Getting Started",
		Permalink: "/blog/getting-started",
		CreatedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		RawMeta: map[string]interface{}{
			"order": 1,
		},
	}
	nvs.PathMap["blog/advanced-topics.md"] = &model.NoteView{
		Path:      "blog/advanced-topics.md",
		Title:     "Advanced Topics",
		Permalink: "/blog/advanced-topics",
		CreatedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
		RawMeta: map[string]interface{}{
			"order": 3,
		},
	}

	return templateviews.NewNVS(nvs, "live")
}

func TestTemplateViews_ByGlobSortByTitle(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/test/blog-list",
		VersionID: 1,
		Path:      "_layouts/test/blog-list.html",
		Content: `<ul>
{{ range i, post := nvs.ByGlob("blog/*.md").SortBy("Title").All() }}
<li>{{ post.Title() }}</li>
{{ end }}
</ul>`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["nvs"] = reflect.ValueOf(createTestNVS())

	var buf bytes.Buffer
	err = layouts.Map["/test/blog-list"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	cupaloy.SnapshotT(t, buf.String())
}

func TestTemplateViews_ByGlobSortByCreatedAtDesc(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/test/blog-recent",
		VersionID: 1,
		Path:      "_layouts/test/blog-recent.html",
		Content: `<ul>
{{ range i, post := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().All() }}
<li>{{ post.Title() }} - {{ post.CreatedAt().Format("2006-01-02") }}</li>
{{ end }}
</ul>`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["nvs"] = reflect.ValueOf(createTestNVS())

	var buf bytes.Buffer
	err = layouts.Map["/test/blog-recent"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	cupaloy.SnapshotT(t, buf.String())
}

func TestTemplateViews_ByGlobSortByMetaLimit(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/test/blog-ordered",
		VersionID: 1,
		Path:      "_layouts/test/blog-ordered.html",
		Content: `<ul>
{{ range i, post := nvs.ByGlob("blog/*.md").SortByMeta("order").Limit(2).All() }}
<li>{{ post.Title() }}</li>
{{ end }}
</ul>`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["nvs"] = reflect.ValueOf(createTestNVS())

	var buf bytes.Buffer
	err = layouts.Map["/test/blog-ordered"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	cupaloy.SnapshotT(t, buf.String())
}

func TestTemplateViews_First(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/test/blog-first",
		VersionID: 1,
		Path:      "_layouts/test/blog-first.html",
		Content:   `{{ post := nvs.ByGlob("blog/*.md").SortBy("Title").First() }}<h1>{{ post.Title() }}</h1>`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["nvs"] = reflect.ValueOf(createTestNVS())

	var buf bytes.Buffer
	err = layouts.Map["/test/blog-first"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	cupaloy.SnapshotT(t, buf.String())
}

// TestJetBlockNamedParams verifies that block parameters with default values work correctly.
// Jet requires default values for parameters, otherwise named arguments don't work.
func TestJetBlockNamedParams(t *testing.T) {
	loader := jet.NewInMemLoader()
	loader.Set("test.html", `{{ block card(title="", body="") }}<div><h1>{{ title }}</h1><p>{{ body }}</p></div>{{ end }}
{{ yield card(title="Hello", body="World") }}`)

	set := jet.NewSet(loader)
	tmpl, err := set.GetTemplate("test.html")
	require.NoError(t, err, "template should parse without error")

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil, nil)
	require.NoError(t, err, "template should execute without error")

	result := buf.String()
	t.Logf("Result: %s", result)
	require.Contains(t, result, "Hello", "title parameter should be passed")
	require.Contains(t, result, "World", "body parameter should be passed")
	require.NotContains(t, result, "false", "should not contain 'false' - params not working")
}

// TestJetBlockContentReserved verifies that "content" is a reserved keyword in Jet.
// Using "content" as a block parameter name causes a parse error.
// In Jet, "content" is used for yielding nested content: {{ yield block() content }}...{{ end }}
func TestJetBlockContentReserved(t *testing.T) {
	loader := jet.NewInMemLoader()
	// "content" as parameter name causes parse error - it's a reserved keyword
	loader.Set("test.html", `{{ block card(title="", content="") }}<div><h1>{{ title }}</h1><p>{{ content }}</p></div>{{ end }}
{{ yield card(title="Hello", content="World") }}`)

	set := jet.NewSet(loader)
	_, err := set.GetTemplate("test.html")

	// Expect parse error: "content" is a reserved keyword
	require.Error(t, err, "template with 'content' parameter should fail to parse")
	require.Contains(t, err.Error(), "content", "error should mention 'content' keyword")
}

// TestJetBlockYieldContent verifies the content wrapping mechanism.
// Use "yield content" inside block to render nested HTML passed via "yield block() content ... end".
func TestJetBlockYieldContent(t *testing.T) {
	loader := jet.NewInMemLoader()
	loader.Set("test.html", `{{ block cta_section(title="") }}
<section>
  <h2>{{ title }}</h2>
  <p>{{ yield content }}</p>
</section>
{{ end }}
{{ yield cta_section(title="Ready to start?") content }}
  Contact us via Telegram.
{{ end }}`)

	set := jet.NewSet(loader)
	tmpl, err := set.GetTemplate("test.html")
	require.NoError(t, err, "template should parse")

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil, nil)
	require.NoError(t, err, "template should execute")

	result := buf.String()
	t.Logf("Result: %s", result)
	require.Contains(t, result, "Ready to start?", "title param should work")
	require.Contains(t, result, "Contact us via Telegram", "content should be yielded")
}

func TestTemplateViews_NoteMeta(t *testing.T) {
	sources := []SourceFile{{
		ID:        "/test/blog-meta",
		VersionID: 1,
		Path:      "_layouts/test/blog-meta.html",
		Content: `<ul>
{{ range i, post := nvs.ByGlob("blog/*.md").SortByMeta("order").All() }}
<li data-order="{{ post.M().GetInt("order", 0) }}">{{ post.Title() }}</li>
{{ end }}
</ul>`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars["nvs"] = reflect.ValueOf(createTestNVS())

	var buf bytes.Buffer
	err = layouts.Map["/test/blog-meta"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	cupaloy.SnapshotT(t, buf.String())
}
