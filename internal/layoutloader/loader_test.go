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
	sources := []model.LayoutSourceFile{{
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
	sources := []model.LayoutSourceFile{{
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
	sources := []model.LayoutSourceFile{{
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
	sources := []model.LayoutSourceFile{{
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
	sources := []model.LayoutSourceFile{{
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
	sources := []model.LayoutSourceFile{{
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

func TestBlockFinder_SimpleBlock(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/test/blocks",
		VersionID: 1,
		Path:      "_layouts/test/blocks.html",
		Content:   `{{ block header(level=1) }}<h{{ level }}>Header</h{{ level }}>{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	// Check block found by name
	block, ok := layouts.Blocks.ByName["header"]
	require.True(t, ok)
	require.Equal(t, "header", block.Name)
	require.Equal(t, "/test/blocks", block.SourceID)
	require.False(t, block.HasContent)

	// Check params with inferred type
	require.Len(t, block.Params, 1)
	require.Equal(t, "level", block.Params[0].Name)
	require.Equal(t, "1", block.Params[0].Default)
	require.Equal(t, "int", block.Params[0].Type)

	// Check block found by full name
	block, ok = layouts.Blocks.ByFullName["/test/blocks#header"]
	require.True(t, ok)
	require.Equal(t, "header", block.Name)
}

func TestBlockFinder_BlockWithContent(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/test/blocks",
		VersionID: 1,
		Path:      "_layouts/test/blocks.html",
		Content:   `{{ block card(title="") }}<div><h1>{{ title }}</h1><p>{{ yield content }}</p></div>{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	block, ok := layouts.Blocks.ByName["card"]
	require.True(t, ok)
	require.True(t, block.HasContent, "block with {{ yield content }} should have HasContent=true")
	require.Equal(t, "string", block.Params[0].Type)
}

func TestBlockFinder_MultipleBlocks(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/test/blocks",
		VersionID: 1,
		Path:      "_layouts/test/blocks.html",
		Content: `{{ block header(level=1) }}<h{{ level }}>Header</h{{ level }}>{{ end }}
{{ block footer() }}<footer>Footer</footer>{{ end }}
{{ block card(title="", subtitle="") }}<div>{{ title }} - {{ subtitle }}</div>{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	_, ok := layouts.Blocks.ByName["header"]
	require.True(t, ok)
	_, ok = layouts.Blocks.ByName["footer"]
	require.True(t, ok)
	_, ok = layouts.Blocks.ByName["card"]
	require.True(t, ok)

	// Check card has 2 params
	card := layouts.Blocks.ByName["card"]
	require.Len(t, card.Params, 2)
	require.Equal(t, "title", card.Params[0].Name)
	require.Equal(t, "subtitle", card.Params[1].Name)
}

func TestBlockFinder_DuplicateBlockNames(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/blocks",
		VersionID: 1,
		Path:      "_layouts/blocks.html",
		Content:   `{{ block card() }}<div>Card from blocks</div>{{ end }}`,
	}, {
		ID:        "/components",
		VersionID: 2,
		Path:      "_layouts/components.html",
		Content:   `{{ block card() }}<div>Card from components</div>{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	// ByName stores last block (components wins)
	card, ok := layouts.Blocks.ByName["card"]
	require.True(t, ok)
	require.Equal(t, "/components", card.SourceID)

	// Each should have unique full name
	_, ok1 := layouts.Blocks.ByFullName["/blocks#card"]
	_, ok2 := layouts.Blocks.ByFullName["/components#card"]
	require.True(t, ok1)
	require.True(t, ok2)
}

func TestBlocksLookup(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/blocks",
		VersionID: 1,
		Path:      "_layouts/blocks.html",
		Content:   `{{ block header() }}Header{{ end }}{{ block card() }}Card1{{ end }}`,
	}, {
		ID:        "/components",
		VersionID: 2,
		Path:      "_layouts/components.html",
		Content:   `{{ block card() }}Card2{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	// Unique block - lookup by name works
	block, found := layouts.Blocks.Lookup("header")
	require.True(t, found)
	require.Equal(t, "header", block.Name)

	// Duplicate block - lookup by name returns last one
	block, found = layouts.Blocks.Lookup("card")
	require.True(t, found)
	require.Equal(t, "/components", block.SourceID)

	// Lookup by full name works
	block, found = layouts.Blocks.Lookup("/blocks#card")
	require.True(t, found)
	require.Equal(t, "/blocks", block.SourceID)

	block, found = layouts.Blocks.Lookup("/components#card")
	require.True(t, found)
	require.Equal(t, "/components", block.SourceID)

	// Non-existent block
	_, found = layouts.Blocks.Lookup("nonexistent")
	require.False(t, found)
}

func TestBlockFinder_ArgTypeMetadata(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/test/blocks",
		VersionID: 1,
		Path:      "_layouts/test/blocks.html",
		Content: `{{ block card(title, subtitle, level=1, featured=false) }}
{{ arg_type("title", "string", "Card title") }}
{{ arg_type("subtitle", "string", "Card subtitle") }}
{{ arg_type("level", "int", "Heading level (1-6)") }}
{{ arg_type("featured", "bool", "Highlight this card") }}
<div class="{{ if featured }}featured{{ end }}">
  <h{{ level }}>{{ title }}</h{{ level }}>
  <p>{{ subtitle }}</p>
</div>
{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	block, ok := layouts.Blocks.ByName["card"]
	require.True(t, ok)
	require.Len(t, block.Params, 4)

	// title - no default, type from arg_type
	require.Equal(t, "title", block.Params[0].Name)
	require.Equal(t, "", block.Params[0].Default)
	require.Equal(t, "string", block.Params[0].Type)
	require.Equal(t, "Card title", block.Params[0].Comment)

	// subtitle - no default, type from arg_type
	require.Equal(t, "subtitle", block.Params[1].Name)
	require.Equal(t, "", block.Params[1].Default)
	require.Equal(t, "string", block.Params[1].Type)
	require.Equal(t, "Card subtitle", block.Params[1].Comment)

	// level - has default, type overridden by arg_type
	require.Equal(t, "level", block.Params[2].Name)
	require.Equal(t, "1", block.Params[2].Default)
	require.Equal(t, "int", block.Params[2].Type)
	require.Equal(t, "Heading level (1-6)", block.Params[2].Comment)

	// featured - has default, type overridden by arg_type
	require.Equal(t, "featured", block.Params[3].Name)
	require.Equal(t, "false", block.Params[3].Default)
	require.Equal(t, "bool", block.Params[3].Type)
	require.Equal(t, "Highlight this card", block.Params[3].Comment)
}

func TestBlockFinder_TypeInferenceFromDefault(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/test/blocks",
		VersionID: 1,
		Path:      "_layouts/test/blocks.html",
		Content: `{{ block test(str="hello", num=42, flt=3.14, flag=true, noDefault) }}
<div>{{ str }} {{ num }} {{ flt }} {{ flag }}</div>
{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	block, ok := layouts.Blocks.ByName["test"]
	require.True(t, ok)
	require.Len(t, block.Params, 5)

	// string default
	require.Equal(t, "str", block.Params[0].Name)
	require.Equal(t, `"hello"`, block.Params[0].Default)
	require.Equal(t, "string", block.Params[0].Type)

	// int default
	require.Equal(t, "num", block.Params[1].Name)
	require.Equal(t, "42", block.Params[1].Default)
	require.Equal(t, "int", block.Params[1].Type)

	// float default
	require.Equal(t, "flt", block.Params[2].Name)
	require.Equal(t, "3.14", block.Params[2].Default)
	require.Equal(t, "float", block.Params[2].Type)

	// bool default
	require.Equal(t, "flag", block.Params[3].Name)
	require.Equal(t, "true", block.Params[3].Default)
	require.Equal(t, "bool", block.Params[3].Type)

	// no default - type unknown
	require.Equal(t, "noDefault", block.Params[4].Name)
	require.Equal(t, "", block.Params[4].Default)
	require.Equal(t, "", block.Params[4].Type)
}

func TestBlockFinder_ArgTypeWithoutComment(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/test/blocks",
		VersionID: 1,
		Path:      "_layouts/test/blocks.html",
		Content: `{{ block simple(name) }}
{{ arg_type("name", "string") }}
<span>{{ name }}</span>
{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	block, ok := layouts.Blocks.ByName["simple"]
	require.True(t, ok)
	require.Len(t, block.Params, 1)

	require.Equal(t, "name", block.Params[0].Name)
	require.Equal(t, "string", block.Params[0].Type)
	require.Equal(t, "", block.Params[0].Comment) // no comment provided
}

func TestBlockFinder_AllBlocksMethod(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/blocks",
		VersionID: 1,
		Path:      "_layouts/blocks.html",
		Content:   `{{ block a() }}A{{ end }}{{ block b() }}B{{ end }}`,
	}, {
		ID:        "/components",
		VersionID: 2,
		Path:      "_layouts/components.html",
		Content:   `{{ block c() }}C{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	all := layouts.Blocks.All()
	require.Len(t, all, 3)

	// Check all blocks are present (order may vary due to map iteration)
	names := make(map[string]bool)
	for _, b := range all {
		names[b.Name] = true
	}
	require.True(t, names["a"])
	require.True(t, names["b"])
	require.True(t, names["c"])
}

func TestBlockFinder_FullName(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:        "/my/blocks",
		VersionID: 1,
		Path:      "_layouts/my/blocks.html",
		Content:   `{{ block header() }}Header{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	block, ok := layouts.Blocks.ByFullName["/my/blocks#header"]
	require.True(t, ok)
	require.Equal(t, "/my/blocks#header", block.FullName())
}

func TestTemplateViews_NoteMeta(t *testing.T) {
	sources := []model.LayoutSourceFile{{
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
