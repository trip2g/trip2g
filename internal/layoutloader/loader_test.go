package layoutloader

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
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

	// Verify rendering works
	var buf bytes.Buffer
	err = layouts.Map["/trip2g/main"].View.Execute(&buf, nil, nil)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<wrapper>hello</wrapper>")
}

// TestYieldBlocksWithParams tests that block parameters work when importing from another file.
func TestYieldBlocksWithParams(t *testing.T) {
	// Note: import path must match ID exactly
	sources := []model.LayoutSourceFile{{
		ID:        "/main",
		VersionID: 1,
		Path:      "_layouts/main.html",
		Content:   `{{ import "/blocks" }}{{ yield card(title="Hello", body="World") }}`,
	}, {
		ID:        "/blocks",
		VersionID: 2,
		Path:      "_layouts/blocks.html",
		Content:   `{{ block card(title="", body="") }}<div><h1>{{ title }}</h1><p>{{ body }}</p></div>{{ end }}`,
	}}

	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	var buf bytes.Buffer
	err = layouts.Map["/main"].View.Execute(&buf, nil, nil)
	require.NoError(t, err)

	t.Logf("Output: %s", buf.String())
	require.Contains(t, buf.String(), "Hello", "title parameter should be passed")
	require.Contains(t, buf.String(), "World", "body parameter should be passed")
}

// TestYieldBlocksWithParams_DirectJet tests with direct Jet usage for comparison.
func TestYieldBlocksWithParams_DirectJet(t *testing.T) {
	// Create custom loader that mimics our jetLoader behavior
	templates := map[string]string{
		"/main":   `{{ import "/blocks" }}{{ yield card(title="Hello", body="World") }}`,
		"/blocks": `{{ block card(title="", body="") }}<div><h1>{{ title }}</h1><p>{{ body }}</p></div>{{ end }}`,
	}

	// Test 1: Same Set for both (like jet.NewInMemLoader)
	t.Run("SameSet", func(t *testing.T) {
		loader := jet.NewInMemLoader()
		for k, v := range templates {
			loader.Set(k, v)
		}
		set := jet.NewSet(loader)
		tmpl, err := set.GetTemplate("/main")
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, nil, nil)
		require.NoError(t, err)
		t.Logf("SameSet output: %s", buf.String())
		require.Contains(t, buf.String(), "Hello")
	})

	// Test 2: New Set for each file (like our loader)
	t.Run("NewSetPerFile", func(t *testing.T) {
		loader := jet.NewInMemLoader()
		for k, v := range templates {
			loader.Set(k, v)
		}

		// Create new Set for /main (mimicking our loader)
		setMain := jet.NewSet(loader)
		tmplMain, err := setMain.GetTemplate("/main")
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmplMain.Execute(&buf, nil, nil)
		require.NoError(t, err)
		t.Logf("NewSetPerFile output: %s", buf.String())
		require.Contains(t, buf.String(), "Hello", "should work even with new set")
	})

	// Test 3: New Set for each file with DevelopmentMode (like our loader)
	t.Run("NewSetPerFile_DevMode", func(t *testing.T) {
		loader := jet.NewInMemLoader()
		for k, v := range templates {
			loader.Set(k, v)
		}

		setMain := jet.NewSet(loader, jet.DevelopmentMode(true))
		tmplMain, err := setMain.GetTemplate("/main")
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmplMain.Execute(&buf, nil, nil)
		require.NoError(t, err)
		t.Logf("NewSetPerFile_DevMode output: %s", buf.String())
		require.Contains(t, buf.String(), "Hello", "should work with dev mode")
	})

	// Test 4: Use custom loader like our jetLoader
	t.Run("CustomLoader", func(t *testing.T) {
		customLoader := &testJetLoader{templates: templates}
		setMain := jet.NewSet(customLoader, jet.DevelopmentMode(true))
		tmplMain, err := setMain.GetTemplate("/main")
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmplMain.Execute(&buf, nil, nil)
		require.NoError(t, err)
		t.Logf("CustomLoader output: %s", buf.String())
		require.Contains(t, buf.String(), "Hello", "should work with custom loader")
	})
}

// testJetLoader mimics our jetLoader for testing.
type testJetLoader struct {
	templates map[string]string
}

func (l *testJetLoader) Exists(templatePath string) bool {
	_, exists := l.templates[templatePath]
	return exists
}

func (l *testJetLoader) Open(templatePath string) (io.ReadCloser, error) {
	content := l.templates[templatePath]
	return io.NopCloser(strings.NewReader(content)), nil
}

// TestYieldBlocksWithParams_MimicLoad mimics exactly what Load() does.
func TestYieldBlocksWithParams_MimicLoad(t *testing.T) {
	templates := map[string]string{
		"/main":   `{{ import "/blocks" }}{{ yield card(title="Hello", body="World") }}`,
		"/blocks": `{{ block card(title="", body="") }}<div><h1>{{ title }}</h1><p>{{ body }}</p></div>{{ end }}`,
	}

	loader := &testJetLoader{templates: templates}

	// Mimic Load() - create new Set for EACH file
	loadedTemplates := make(map[string]*jet.Template)

	for id := range templates {
		// Each file gets its own Set (like our loader does)
		set := jet.NewSet(loader, jet.DevelopmentMode(true))
		tmpl, err := set.GetTemplate(id)
		require.NoError(t, err)
		loadedTemplates[id] = tmpl
		t.Logf("Loaded template %s", id)
	}

	// Execute /main
	var buf bytes.Buffer
	err := loadedTemplates["/main"].Execute(&buf, nil, nil)
	require.NoError(t, err)

	t.Logf("Output: %s", buf.String())
	require.Contains(t, buf.String(), "Hello", "title parameter should be passed")
}

// TestYieldNodeParametersPreserved verifies that our fixed assetFinder
// doesn't clear YieldNode parameters (regression test for the bug fix).
func TestYieldNodeParametersPreserved(t *testing.T) {
	templates := map[string]string{
		"/main":   `{{ import "/blocks" }}{{ yield card(title="Hello", body="World") }}`,
		"/blocks": `{{ block card(title="", body="") }}<div><h1>{{ title }}</h1><p>{{ body }}</p></div>{{ end }}`,
	}

	loader := &testJetLoader{templates: templates}
	set := jet.NewSet(loader, jet.DevelopmentMode(true))
	tmpl, err := set.GetTemplate("/main")
	require.NoError(t, err)

	// Before walk - params should work
	var buf1 bytes.Buffer
	err = tmpl.Execute(&buf1, nil, nil)
	require.NoError(t, err)
	t.Logf("Before walk: %s", buf1.String())
	require.Contains(t, buf1.String(), "Hello", "params should work before walk")

	// Apply our FIXED assetFinder walk
	fixedWalker := &assetFinder{}
	utils.Walk(tmpl, fixedWalker)

	// After walk with fixed code - params should STILL work
	var buf2 bytes.Buffer
	err = tmpl.Execute(&buf2, nil, nil)
	require.NoError(t, err)
	t.Logf("After walk (fixed): %s", buf2.String())
	require.Contains(t, buf2.String(), "Hello", "params should still work after walk with fixed code")
}

// TestYieldNodeParametersCleared_BugDemo demonstrates that unconditionally
// setting Parameters clears existing params (documents why we need the nil check).
func TestYieldNodeParametersCleared_BugDemo(t *testing.T) {
	templates := map[string]string{
		"/main":   `{{ import "/blocks" }}{{ yield card(title="Hello", body="World") }}`,
		"/blocks": `{{ block card(title="", body="") }}<div><h1>{{ title }}</h1><p>{{ body }}</p></div>{{ end }}`,
	}

	loader := &testJetLoader{templates: templates}
	set := jet.NewSet(loader, jet.DevelopmentMode(true))
	tmpl, err := set.GetTemplate("/main")
	require.NoError(t, err)

	// Before walk - params work
	var buf1 bytes.Buffer
	err = tmpl.Execute(&buf1, nil, nil)
	require.NoError(t, err)
	require.Contains(t, buf1.String(), "Hello")

	// Apply buggy walker that unconditionally clears Parameters
	buggyWalker := &buggyAssetFinder{}
	utils.Walk(tmpl, buggyWalker)

	// After buggy walk - params are CLEARED (expected - demonstrating the bug)
	var buf2 bytes.Buffer
	err = tmpl.Execute(&buf2, nil, nil)
	require.NoError(t, err)
	t.Logf("After buggy walk: %s", buf2.String())
	require.NotContains(t, buf2.String(), "Hello", "buggy walker should clear params (demonstrating the bug)")
	require.Contains(t, buf2.String(), "<h1></h1>", "params should be empty after buggy walk")
}

// buggyAssetFinder reproduces the original bug (unconditional Parameters assignment).
type buggyAssetFinder struct{}

func (w *buggyAssetFinder) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}

	// This is the BUG - unconditionally clearing Parameters
	if yieldNode, ok := node.(*jet.YieldNode); ok {
		yieldNode.Parameters = &jet.BlockParameterList{}
	}

	vc.Visit(node)
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

// TestJetBlockNamedParams_CrossFile tests if Jet itself supports params with import.
// This isolates whether the bug is in Jet or in our loader.
func TestJetBlockNamedParams_CrossFile(t *testing.T) {
	loader := jet.NewInMemLoader()
	loader.Set("blocks.html", `{{ block card(title="", body="") }}<div><h1>{{ title }}</h1><p>{{ body }}</p></div>{{ end }}`)
	loader.Set("main.html", `{{ import "blocks.html" }}{{ yield card(title="Hello", body="World") }}`)

	set := jet.NewSet(loader)
	tmpl, err := set.GetTemplate("main.html")
	require.NoError(t, err, "template should parse without error")

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil, nil)
	require.NoError(t, err, "template should execute without error")

	result := buf.String()
	t.Logf("Result: %s", result)
	require.Contains(t, result, "Hello", "title parameter should be passed")
	require.Contains(t, result, "World", "body parameter should be passed")
}

// TestJetBlockContentReserved verifies that "content" is a reserved keyword in Jet.
// Using "content" as a block parameter name causes a parse error.
// In Jet, "content" is used for yielding nested content: {{ yield block() content }}...{{ end }}.
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
	require.Empty(t, block.Params[0].Default)
	require.Equal(t, "string", block.Params[0].Type)
	require.Equal(t, "Card title", block.Params[0].Comment)

	// subtitle - no default, type from arg_type
	require.Equal(t, "subtitle", block.Params[1].Name)
	require.Empty(t, block.Params[1].Default)
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
	require.Empty(t, block.Params[4].Default)
	require.Empty(t, block.Params[4].Type)
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
	require.Empty(t, block.Params[0].Comment) // no comment provided
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

// TestImportBlocksOrder verifies that file order in sourceFiles doesn't affect
// import/block resolution. Both orders should work identically.
func TestImportBlocksOrder(t *testing.T) {
	indexContent := `{{ import "/blocks" }}<html>{{ yield header(title="Welcome") }}</html>`

	blocksContent := `{{ block header(title="") }}<h1>{{ title }}</h1>{{ end }}`

	// Order 1: index first, then blocks
	sourcesIndexFirst := []model.LayoutSourceFile{
		{ID: "/index", VersionID: 1, Path: "_layouts/index.html", Content: indexContent},
		{ID: "/blocks", VersionID: 2, Path: "_layouts/blocks.html", Content: blocksContent},
	}

	// Order 2: blocks first, then index (reversed)
	sourcesBlocksFirst := []model.LayoutSourceFile{
		{ID: "/blocks", VersionID: 2, Path: "_layouts/blocks.html", Content: blocksContent},
		{ID: "/index", VersionID: 1, Path: "_layouts/index.html", Content: indexContent},
	}

	env := &testEnv{logger: &logger.TestLogger{}}

	// Test order 1: index first
	layouts1, err := Load(env, sourcesIndexFirst, Options{})
	require.NoError(t, err, "should load with index first")

	var buf1 bytes.Buffer
	err = layouts1.Map["/index"].View.Execute(&buf1, nil, nil)
	require.NoError(t, err, "should execute index template (order 1)")
	require.Contains(t, buf1.String(), "<h1>Welcome</h1>", "block should render correctly (order 1)")

	// Test order 2: blocks first (reversed)
	layouts2, err := Load(env, sourcesBlocksFirst, Options{})
	require.NoError(t, err, "should load with blocks first")

	var buf2 bytes.Buffer
	err = layouts2.Map["/index"].View.Execute(&buf2, nil, nil)
	require.NoError(t, err, "should execute index template (order 2)")
	require.Contains(t, buf2.String(), "<h1>Welcome</h1>", "block should render correctly (order 2)")

	// Both should produce identical output
	require.Equal(t, buf1.String(), buf2.String(), "output should be identical regardless of file order")
}

// TestImportBlocksOrder_NestedImport tests nested imports work regardless of order.
func TestImportBlocksOrder_NestedImport(t *testing.T) {
	// index imports components, components imports blocks
	indexContent := `{{ import "/components" }}{{ yield page() }}`
	componentsContent := `{{ import "/blocks" }}{{ block page() }}<div>{{ yield btn(text="Click") }}</div>{{ end }}`
	blocksContent := `{{ block btn(text="") }}<button>{{ text }}</button>{{ end }}`

	// Try different orderings
	orderings := [][]model.LayoutSourceFile{
		// Order 1: index, components, blocks
		{
			{ID: "/index", VersionID: 1, Path: "_layouts/index.html", Content: indexContent},
			{ID: "/components", VersionID: 2, Path: "_layouts/components.html", Content: componentsContent},
			{ID: "/blocks", VersionID: 3, Path: "_layouts/blocks.html", Content: blocksContent},
		},
		// Order 2: blocks, components, index (fully reversed)
		{
			{ID: "/blocks", VersionID: 3, Path: "_layouts/blocks.html", Content: blocksContent},
			{ID: "/components", VersionID: 2, Path: "_layouts/components.html", Content: componentsContent},
			{ID: "/index", VersionID: 1, Path: "_layouts/index.html", Content: indexContent},
		},
		// Order 3: blocks, index, components (mixed)
		{
			{ID: "/blocks", VersionID: 3, Path: "_layouts/blocks.html", Content: blocksContent},
			{ID: "/index", VersionID: 1, Path: "_layouts/index.html", Content: indexContent},
			{ID: "/components", VersionID: 2, Path: "_layouts/components.html", Content: componentsContent},
		},
	}

	env := &testEnv{logger: &logger.TestLogger{}}
	var results []string

	for i, sources := range orderings {
		layouts, err := Load(env, sources, Options{})
		require.NoError(t, err, "order %d: should load", i+1)

		var buf bytes.Buffer
		err = layouts.Map["/index"].View.Execute(&buf, nil, nil)
		require.NoError(t, err, "order %d: should execute", i+1)
		require.Contains(t, buf.String(), "<button>Click</button>", "order %d: nested block should render", i+1)

		results = append(results, buf.String())
	}

	// All orderings should produce identical output
	for i := 1; i < len(results); i++ {
		require.Equal(t, results[0], results[i], "order %d should match order 1", i+1)
	}
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

func TestTemplateViews_NestedSections(t *testing.T) {
	// Create a note with nested headings (h2 categories, h3 slides)
	noteContent := []byte(`
## Category 1

Introduction to category 1.

### Slide 1.1

Content for slide 1.1

### Slide 1.2

Content for slide 1.2

## Category 2

Introduction to category 2.

### Slide 2.1

Content for slide 2.1
`)

	// Load note through mdloader to get proper PartialRenderer
	log := &logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "presentation.md",
			Content: noteContent,
		}},
		Log: log,
	})
	require.NoError(t, err)
	require.Len(t, pages.List, 1)

	noteView := pages.List[0]

	// Template that uses nested Sections
	sources := []model.LayoutSourceFile{{
		ID:        "/test/nested-sections",
		VersionID: 1,
		Path:      "_layouts/test/nested-sections.html",
		Content: `<div class="presentation">
{{ range idx, category := note.PartialRenderer().Sections(2) }}
<section class="category" data-title="{{ category.Title }}">
  <h2>{{ category.TitleHTML | unsafe }}</h2>
{{ range slideIdx, slide := category.Sections(3) }}
  <section class="slide" data-slide-num="{{ slideIdx + 1 }}" data-category="{{ category.Title }}">
    <h3>{{ slide.TitleHTML | unsafe }}</h3>
    {{ slide.ContentHTML | unsafe }}
  </section>
{{ end }}
</section>
{{ end }}
</div>`,
	}}

	env := &testEnv{logger: log}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	// Create templateviews.Note wrapper
	nvs := model.NewNoteViews()
	nvs.RegisterNote(noteView)
	tplNVS := templateviews.NewNVS(nvs, "live")
	note := tplNVS.ByPath("presentation.md")
	require.NotNil(t, note)

	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(note)

	var buf bytes.Buffer
	err = layouts.Map["/test/nested-sections"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	result := buf.String()
	t.Logf("Result:\n%s", result)

	// Verify category 1 and its slides
	require.Contains(t, result, `data-title="Category 1"`)
	require.Contains(t, result, `data-slide-num="1" data-category="Category 1"`)
	require.Contains(t, result, `data-slide-num="2" data-category="Category 1"`)
	require.Contains(t, result, "Slide 1.1")
	require.Contains(t, result, "Slide 1.2")
	require.Contains(t, result, "Content for slide 1.1")
	require.Contains(t, result, "Content for slide 1.2")

	// Verify category 2 and its slides
	require.Contains(t, result, `data-title="Category 2"`)
	require.Contains(t, result, `data-slide-num="1" data-category="Category 2"`)
	require.Contains(t, result, "Slide 2.1")
	require.Contains(t, result, "Content for slide 2.1")

	cupaloy.SnapshotT(t, result)
}

func TestTemplateViews_NestedSectionByTitle(t *testing.T) {
	// Create a note with nested headings
	noteContent := []byte(`
## Features

### Performance

Our app is fast.

### Security

Our app is secure.

## Pricing

### Basic Plan

$10/month
`)

	// Load note through mdloader to get proper PartialRenderer
	log := &logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "features.md",
			Content: noteContent,
		}},
		Log: log,
	})
	require.NoError(t, err)
	require.Len(t, pages.List, 1)

	noteView := pages.List[0]

	// Template that uses Section(title) to find specific subsection
	sources := []model.LayoutSourceFile{{
		ID:        "/test/section-by-title",
		VersionID: 1,
		Path:      "_layouts/test/section-by-title.html",
		Content: `{{ features := note.PartialRenderer().Section("Features") }}
{{ if features }}
<div class="features">
  {{ security := features.Section("Security") }}
  {{ if security }}
  <div class="security-highlight">
    <h4>{{ security.TitleHTML | unsafe }}</h4>
    {{ security.ContentHTML | unsafe }}
  </div>
  {{ end }}
</div>
{{ end }}`,
	}}

	env := &testEnv{logger: log}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)

	// Create templateviews.Note wrapper
	nvs := model.NewNoteViews()
	nvs.RegisterNote(noteView)
	tplNVS := templateviews.NewNVS(nvs, "live")
	note := tplNVS.ByPath("features.md")
	require.NotNil(t, note)

	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(note)

	var buf bytes.Buffer
	err = layouts.Map["/test/section-by-title"].View.Execute(&buf, vars, nil)
	require.NoError(t, err)

	result := buf.String()
	t.Logf("Result:\n%s", result)

	// Verify that nested Section(title) works
	require.Contains(t, result, "security-highlight")
	require.Contains(t, result, "Security")
	require.Contains(t, result, "Our app is secure")

	// Should NOT contain Performance (we only requested Security)
	require.NotContains(t, result, "Performance")
}
