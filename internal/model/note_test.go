package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestPerparePermalink(t *testing.T) {
	n := NoteView{Path: "/Моя заметка + другая заметка.md"}
	n.PreparePermalink()

	require.Equal(t, "/moya_zametka_drugaya_zametka", n.Permalink)

	n.Path = "Моя особая + страница"
	n.PreparePermalink()

	require.Equal(t, "/moya_osobaya_stranica", n.Permalink)

	n.Path = "_banner.md"
	n.PreparePermalink()

	require.Equal(t, "/_banner", n.Permalink)
	require.False(t, n.IsIndex)

	n.Path = "/nested/index.md"
	n.PreparePermalink()

	require.Equal(t, "/nested", n.Permalink)
	require.True(t, n.IsIndex)

	n.Path = "/nested/_index.md"
	n.PreparePermalink()

	require.Equal(t, "/nested", n.Permalink)
	require.True(t, n.IsIndex)
}

func TestPreparePermalinkWithSlug(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		slug              string
		expectedPermalink string
		expectedOriginal  string
		expectedIsIndex   bool
	}{
		{
			name:              "relative slug replaces filename",
			path:              "my-file.md",
			slug:              "custom-name",
			expectedPermalink: "/custom-name",
			expectedOriginal:  "/custom-name",
			expectedIsIndex:   false,
		},
		{
			name:              "relative slug in nested folder",
			path:              "folder/my-file.md",
			slug:              "my-custom-page",
			expectedPermalink: "/folder/my-custom-page",
			expectedOriginal:  "/folder/my-custom-page",
			expectedIsIndex:   false,
		},
		{
			name:              "absolute slug overrides full path",
			path:              "some/deep/path/file.md",
			slug:              "/archive/old-post",
			expectedPermalink: "/archive/old-post",
			expectedOriginal:  "/archive/old-post",
			expectedIsIndex:   false,
		},
		{
			name:              "relative slug with subdirectory",
			path:              "root-file.md",
			slug:              "sub/nested/page",
			expectedPermalink: "/sub/nested/page",
			expectedOriginal:  "/sub/nested/page",
			expectedIsIndex:   false,
		},
		{
			name:              "cyrillic slug no transliteration",
			path:              "file.md",
			slug:              "моя-страница",
			expectedPermalink: "/%D0%BC%D0%BE%D1%8F-%D1%81%D1%82%D1%80%D0%B0%D0%BD%D0%B8%D1%86%D0%B0",
			expectedOriginal:  "/моя-страница",
			expectedIsIndex:   false,
		},
		{
			name:              "slug with spaces URL encoded",
			path:              "file.md",
			slug:              "page with spaces",
			expectedPermalink: "/page%20with%20spaces",
			expectedOriginal:  "/page with spaces",
			expectedIsIndex:   false,
		},
		{
			name:              "absolute slug with index",
			path:              "file.md",
			slug:              "/section/index",
			expectedPermalink: "/section/index",
			expectedOriginal:  "/section/index",
			expectedIsIndex:   true,
		},
		{
			name:              "relative slug ending with index",
			path:              "folder/file.md",
			slug:              "sub/index",
			expectedPermalink: "/folder/sub/index",
			expectedOriginal:  "/folder/sub/index",
			expectedIsIndex:   true,
		},
		{
			name:              "empty slug uses default behavior",
			path:              "Тестовый файл.md",
			slug:              "",
			expectedPermalink: "/testovyij_fajl",
			expectedOriginal:  "/тестовый_файл",
			expectedIsIndex:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoteView{
				Path: tt.path,
				Slug: tt.slug,
			}

			n.PreparePermalink()

			require.Equal(t, tt.expectedPermalink, n.Permalink)
			require.Equal(t, tt.expectedOriginal, n.PermalinkOriginal)
			require.Equal(t, tt.expectedIsIndex, n.IsIndex)
		})
	}
}

func TestExtractReadingTime(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		rawMeta     map[string]interface{}
		expectedMin int
	}{
		{
			name:        "empty content",
			content:     "",
			rawMeta:     make(map[string]interface{}),
			expectedMin: 0,
		},
		{
			name:        "short content",
			content:     "Hello world",
			rawMeta:     make(map[string]interface{}),
			expectedMin: 1, // minimum is 1 minute
		},
		{
			name: "content with markdown",
			content: `# Header
This is a **bold** text with [link](https://example.com) and [[wikilink|display text]].

` + "```go\ncode block\n```" + `

- List item 1
- List item 2

> Blockquote text

Some more text to reach word count.`,
			rawMeta:     make(map[string]interface{}),
			expectedMin: 1,
		},
		{
			name:        "meta override as int",
			content:     "Some content here",
			rawMeta:     map[string]interface{}{"reading_time": 5},
			expectedMin: 5,
		},
		{
			name:        "meta override as float",
			content:     "Some content here",
			rawMeta:     map[string]interface{}{"reading_time": 3.0},
			expectedMin: 3,
		},
		{
			name:        "meta override as string",
			content:     "Some content here",
			rawMeta:     map[string]interface{}{"reading_time": "7"},
			expectedMin: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoteView{
				Content: []byte(tt.content),
				RawMeta: tt.rawMeta,
			}

			n.extractReadingTime()

			require.Equal(t, tt.expectedMin, n.ReadingTime)
		})
	}
}

func TestExtractReadingComplexity(t *testing.T) {
	tests := []struct {
		name         string
		rawMeta      map[string]interface{}
		expectedComp int
		expectError  bool
	}{
		{
			name:         "no complexity meta",
			rawMeta:      make(map[string]interface{}),
			expectedComp: 0, // default is easy
			expectError:  false,
		},
		{
			name:         "complexity as int 0",
			rawMeta:      map[string]interface{}{"complexity": 0},
			expectedComp: 0,
			expectError:  false,
		},
		{
			name:         "complexity as int 1",
			rawMeta:      map[string]interface{}{"complexity": 1},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as int 2",
			rawMeta:      map[string]interface{}{"complexity": 2},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "complexity as float 1.0",
			rawMeta:      map[string]interface{}{"complexity": 1.0},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as string easy",
			rawMeta:      map[string]interface{}{"complexity": "easy"},
			expectedComp: 0,
			expectError:  false,
		},
		{
			name:         "complexity as string medium",
			rawMeta:      map[string]interface{}{"complexity": "medium"},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as string hard",
			rawMeta:      map[string]interface{}{"complexity": "hard"},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "complexity as string 0",
			rawMeta:      map[string]interface{}{"complexity": "0"},
			expectedComp: 0,
			expectError:  false,
		},
		{
			name:         "complexity as string 1",
			rawMeta:      map[string]interface{}{"complexity": "1"},
			expectedComp: 1,
			expectError:  false,
		},
		{
			name:         "complexity as string 2",
			rawMeta:      map[string]interface{}{"complexity": "2"},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "reading_complexity key",
			rawMeta:      map[string]interface{}{"reading_complexity": "hard"},
			expectedComp: 2,
			expectError:  false,
		},
		{
			name:         "invalid int complexity",
			rawMeta:      map[string]interface{}{"complexity": 5},
			expectedComp: 0,
			expectError:  true,
		},
		{
			name:         "invalid string complexity",
			rawMeta:      map[string]interface{}{"complexity": "invalid"},
			expectedComp: 0,
			expectError:  true,
		},
		{
			name:         "invalid type complexity",
			rawMeta:      map[string]interface{}{"complexity": []string{"easy"}},
			expectedComp: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoteView{
				RawMeta: tt.rawMeta,
			}

			err := n.extractReadingComplexity()

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expectedComp, n.ReadingComplexity)
		})
	}
}

func TestExtractHeadings(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedHeadings NoteViewHeadings
	}{
		{
			name:             "no headings",
			content:          "Just some regular text content.",
			expectedHeadings: nil,
		},
		{
			name: "single heading",
			content: `# Main Title
Some content here.`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Main Title", Level: 1, ID: "main_title_0"},
			},
		},
		{
			name: "multiple headings different levels",
			content: `# Chapter 1
Some intro text.

## Section 1.1
More content.

### Subsection 1.1.1
Even more content.

## Section 1.2
Final content.`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Chapter 1", Level: 1, ID: "chapter_1_0"},
				{Text: "Section 1.1", Level: 2, ID: "section_1_1_0"},
				{Text: "Subsection 1.1.1", Level: 3, ID: "subsection_1_1_1_0"},
				{Text: "Section 1.2", Level: 2, ID: "section_1_2_0"},
			},
		},
		{
			name: "headings with formatting",
			content: `# **Bold** Heading
## *Italic* Heading
### [Link](http://example.com) Heading`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Bold Heading", Level: 1, ID: "bold_heading_0"},
				{Text: "Italic Heading", Level: 2, ID: "italic_heading_0"},
				{Text: "Link Heading", Level: 3, ID: "link_heading_0"},
			},
		},
		{
			name: "headings with gaps get normalized",
			content: `## Second Level
#### Fourth Level
###### Sixth Level`,
			expectedHeadings: NoteViewHeadings{
				{Text: "Second Level", Level: 1, ID: "second_level_0"},
				{Text: "Fourth Level", Level: 2, ID: "fourth_level_0"},
				{Text: "Sixth Level", Level: 3, ID: "sixth_level_0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to parse the markdown to create an AST
			parser := goldmark.New()
			doc := parser.Parser().Parse(text.NewReader([]byte(tt.content)))

			n := &NoteView{
				Content: []byte(tt.content),
				ast:     doc,
			}

			n.extractHeadingsAndGenerateIDs()

			require.Equal(t, tt.expectedHeadings, n.Headings)
		})
	}
}

func TestNoteViewHeadings_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    NoteViewHeadings
		expected NoteViewHeadings
	}{
		{
			name:     "empty headings",
			input:    NoteViewHeadings{},
			expected: NoteViewHeadings{},
		},
		{
			name: "already normalized levels 1,2,3",
			input: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H2", Level: 2, ID: "h2"},
				{Text: "H3", Level: 3, ID: "h3"},
			},
			expected: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H2", Level: 2, ID: "h2"},
				{Text: "H3", Level: 3, ID: "h3"},
			},
		},
		{
			name: "only level 2 becomes level 1",
			input: NoteViewHeadings{
				{Text: "H2a", Level: 2, ID: "h2a"},
				{Text: "H2b", Level: 2, ID: "h2b"},
			},
			expected: NoteViewHeadings{
				{Text: "H2a", Level: 1, ID: "h2a"},
				{Text: "H2b", Level: 1, ID: "h2b"},
			},
		},
		{
			name: "levels 2 and 6 become 1 and 2",
			input: NoteViewHeadings{
				{Text: "H2", Level: 2, ID: "h2"},
				{Text: "H6a", Level: 6, ID: "h6a"},
				{Text: "H6b", Level: 6, ID: "h6b"},
			},
			expected: NoteViewHeadings{
				{Text: "H2", Level: 1, ID: "h2"},
				{Text: "H6a", Level: 2, ID: "h6a"},
				{Text: "H6b", Level: 2, ID: "h6b"},
			},
		},
		{
			name: "levels 1,3,5 become 1,2,3",
			input: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H3", Level: 3, ID: "h3"},
				{Text: "H5", Level: 5, ID: "h5"},
				{Text: "H3b", Level: 3, ID: "h3b"},
			},
			expected: NoteViewHeadings{
				{Text: "H1", Level: 1, ID: "h1"},
				{Text: "H3", Level: 2, ID: "h3"},
				{Text: "H5", Level: 3, ID: "h5"},
				{Text: "H3b", Level: 2, ID: "h3b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy since Normalize modifies in place
			input := make(NoteViewHeadings, len(tt.input))
			copy(input, tt.input)

			input.Normalize()

			require.Equal(t, tt.expected, input)
		})
	}
}

func TestParseRoute(t *testing.T) {
	cases := []struct {
		input    string
		wantHost string
		wantPath string
	}{
		{"/about", "", "/about"},
		{"/", "", "/"},
		{"foo.com", "foo.com", ""},
		{"foo.com/", "foo.com", "/"},
		{"foo.com/hello", "foo.com", "/hello"},
		{"www.foo.com/", "foo.com", "/"},
		{"FOO.COM/", "foo.com", "/"},
		{"localhost:8081/path", "localhost:8081", "/path"},
		{"  /about  ", "", "/about"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			r := ParseRoute(tc.input)
			require.Equal(t, tc.wantHost, r.Host)
			require.Equal(t, tc.wantPath, r.Path)
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	require.Equal(t, "foo.com", NormalizeDomain("www.foo.com"))
	require.Equal(t, "foo.com", NormalizeDomain("FOO.COM"))
	require.Equal(t, "foo.com", NormalizeDomain("  foo.com  "))
	require.Equal(t, "foo.com", NormalizeDomain("www.FOO.COM"))
}

func TestExtractRoutes_Single(t *testing.T) {
	note := &NoteView{
		RawMeta: map[string]interface{}{
			"route": "/about",
		},
	}
	routes := note.ExtractRoutes()
	require.Len(t, routes, 1)
	require.Empty(t, routes[0].Host)
	require.Equal(t, "/about", routes[0].Path)
}

func TestExtractRoutes_Multiple(t *testing.T) {
	note := &NoteView{
		RawMeta: map[string]interface{}{
			"routes": []interface{}{"/alias", "foo.com/"},
		},
	}
	routes := note.ExtractRoutes()
	require.Len(t, routes, 2)
	require.Empty(t, routes[0].Host)
	require.Equal(t, "/alias", routes[0].Path)
	require.Equal(t, "foo.com", routes[1].Host)
	require.Equal(t, "/", routes[1].Path)
}

func TestExtractRoutes_Mixed(t *testing.T) {
	note := &NoteView{
		RawMeta: map[string]interface{}{
			"route":  "/my-alias",
			"routes": []interface{}{"foo.com/", "bar.com/landing"},
		},
	}
	routes := note.ExtractRoutes()
	require.Len(t, routes, 3)
}

func TestExtractRoutes_Empty(t *testing.T) {
	note := &NoteView{
		RawMeta: map[string]interface{}{},
	}
	routes := note.ExtractRoutes()
	require.Empty(t, routes)
}

func TestExtractRoutes_Deduplication(t *testing.T) {
	note := &NoteView{
		RawMeta: map[string]interface{}{
			"route":  "/about",
			"routes": []interface{}{"/about"},
		},
	}
	routes := note.ExtractRoutes()
	require.Len(t, routes, 1)
}

func TestRouteMap_Registration(t *testing.T) {
	nvs := NewNoteViews()
	note := &NoteView{
		Permalink:         "/my-note",
		PermalinkOriginal: "/my-note",
		Path:              "my-note.md",
		Routes: []ParsedRoute{
			{Host: "", Path: "/alias"},
			{Host: "foo.com", Path: "/"},
		},
	}
	nvs.RegisterNote(note)
	require.Equal(t, note, nvs.RouteMap[""]["/alias"])
	require.Equal(t, note, nvs.RouteMap["foo.com"]["/"])
}

func TestRouteMap_EmptyPath_UsesPermalink(t *testing.T) {
	nvs := NewNoteViews()
	note := &NoteView{
		Permalink:         "/my-note",
		PermalinkOriginal: "/my-note",
		Path:              "my-note.md",
		Routes: []ParsedRoute{
			{Host: "foo.com", Path: ""},
		},
	}
	nvs.RegisterNote(note)
	// When Path is "", the note's Permalink is used as the key
	require.Equal(t, note, nvs.RouteMap["foo.com"]["/my-note"])
}

func TestGetByRoute(t *testing.T) {
	nvs := NewNoteViews()
	note := &NoteView{
		Permalink:         "/note",
		PermalinkOriginal: "/note",
		Path:              "note.md",
		Routes:            []ParsedRoute{{Host: "foo.com", Path: "/"}},
	}
	nvs.RegisterNote(note)

	require.Equal(t, note, nvs.GetByRoute("foo.com", "/"))
	require.Nil(t, nvs.GetByRoute("bar.com", "/"))
	require.Nil(t, nvs.GetByRoute("foo.com", "/other"))
}

func TestCustomDomains(t *testing.T) {
	nvs := NewNoteViews()
	note := &NoteView{
		Permalink:         "/note",
		PermalinkOriginal: "/note",
		Path:              "note.md",
		Routes: []ParsedRoute{
			{Host: "", Path: "/alias"},   // main domain
			{Host: "foo.com", Path: "/"}, // custom
			{Host: "bar.com", Path: "/"}, // custom
		},
	}
	nvs.RegisterNote(note)
	domains := nvs.CustomDomains()
	require.Len(t, domains, 2)
	require.Contains(t, domains, "foo.com")
	require.Contains(t, domains, "bar.com")
}

func TestSlugUnchanged(t *testing.T) {
	// slug changes Permalink, route does NOT affect nv.Map
	nvs := NewNoteViews()
	note := &NoteView{
		Path: "my-note.md",
		Slug: "/custom-url",
		RawMeta: map[string]interface{}{
			"route": "/alias",
		},
	}
	note.PreparePermalink()
	note.Routes = note.ExtractRoutes()
	nvs.RegisterNote(note)

	// slug changed the permalink
	require.Equal(t, "/custom-url", note.Permalink)
	// note is in nv.Map under /custom-url
	require.Equal(t, note, nvs.Map["/custom-url"])
	// route alias is in RouteMap, not nv.Map
	require.Equal(t, note, nvs.RouteMap[""]["/alias"])
	require.Nil(t, nvs.Map["/alias"])
}

func TestNoCollision_IndexAndRouteRoot(t *testing.T) {
	// A note with route: / does NOT overwrite nv.Map["/"] from _index.md
	nvs := NewNoteViews()

	indexNote := &NoteView{
		Path:              "_index.md",
		Permalink:         "/",
		PermalinkOriginal: "/",
		IsIndex:           true,
	}
	nvs.RegisterNote(indexNote)

	routeNote := &NoteView{
		Path:              "about.md",
		Permalink:         "/about",
		PermalinkOriginal: "/about",
		Routes:            []ParsedRoute{{Host: "", Path: "/"}},
	}
	nvs.RegisterNote(routeNote)

	// nv.Map["/"] is still _index.md (route doesn't touch Map)
	require.Equal(t, indexNote, nvs.Map["/"])
	// But RouteMap[""]["/"] = routeNote (route wins for RouteMap)
	require.Equal(t, routeNote, nvs.RouteMap[""]["/"])
}

func TestNoteViewsRegisterRegularNote(t *testing.T) {
	nv := NoteViews{
		Map:     map[string]*NoteView{},
		PathMap: map[string]*NoteView{},
	}

	note := &NoteView{
		Path:              "hello world.md",
		Permalink:         "/hello_world",
		PermalinkOriginal: "/hello world",
	}

	nv.RegisterNote(note)

	require.Equal(t, map[string]*NoteView{"/hello_world": note, "/hello world": note}, nv.Map)
	require.Equal(t, map[string]*NoteView{"hello world.md": note}, nv.PathMap)
}

func TestNoteViewsRegisterIndexNote(t *testing.T) {
	nv := NoteViews{
		Map:     map[string]*NoteView{},
		PathMap: map[string]*NoteView{},
	}

	note := &NoteView{
		IsIndex:           true,
		Path:              "hello world/index.md",
		Permalink:         "/hello_world",
		PermalinkOriginal: "/hello world",
	}

	nv.RegisterNote(note)

	require.Equal(t, map[string]*NoteView{"hello world/index.md": note}, nv.PathMap)
	require.Equal(t, map[string]*NoteView{
		"/hello_world":       note,
		"/hello world":       note,
		"/hello_world/index": note,
	}, nv.Map)
}
