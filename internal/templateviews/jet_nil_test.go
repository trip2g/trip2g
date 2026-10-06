package templateviews_test

import (
	"bytes"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

func newJetTestNVS() *templateviews.NVS {
	nvs := model.NewNoteViews()

	about := &model.NoteView{
		Path:      "about.md",
		Permalink: "/about",
		Title:     "About",
		RawMeta:   map[string]interface{}{"subtitle": "Who we are"},
	}
	nvs.PathMap["about.md"] = about
	nvs.Map["/about"] = about
	nvs.BasenameMap = map[string][]*model.NoteView{"about": {about}}

	post := &model.NoteView{Path: "blog/post.md", Permalink: "/blog/post", Title: "Post"}
	nvs.PathMap["blog/post.md"] = post
	nvs.Map["/blog/post"] = post

	return templateviews.NewNVS(nvs, "live")
}

func renderJet(t *testing.T, source string) (string, error) {
	t.Helper()

	set := jet.NewSet(jet.NewInMemLoader())

	tpl, err := set.Parse("test.jet", source)
	if err != nil {
		return "", err
	}

	vars := make(jet.VarMap)
	vars.Set("nvs", newJetTestNVS())

	var out bytes.Buffer
	err = tpl.Execute(&out, vars, nil)

	return out.String(), err
}

func TestJetLookupNilChecks(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "ByPath miss",
			source:   `{{ x := nvs.ByPath("/missing.md") }}{{ if x == nil }}A{{ end }}{{ if !x }}B{{ end }}{{ if x }}C{{ end }}`,
			expected: "AB",
		},
		{
			name:     "ByPath hit",
			source:   `{{ x := nvs.ByPath("/about.md") }}{{ if x == nil }}A{{ end }}{{ if !x }}B{{ end }}{{ if x }}C{{ x.Title() }}{{ end }}`,
			expected: "CAbout",
		},
		{
			name:     "ByPermalink miss and hit",
			source:   `{{ nvs.ByPermalink("/missing") == nil }} {{ nvs.ByPermalink("/about") == nil }}`,
			expected: "true false",
		},
		{
			name:     "ByWikilink miss and hit",
			source:   `{{ nvs.ByWikilink("missing") == nil }} {{ nvs.ByWikilink("about").Title() }}`,
			expected: "true About",
		},
		{
			name:     "First miss and hit",
			source:   `{{ nvs.ByGlob("none/*.md").First() == nil }} {{ nvs.ByGlob("blog/*.md").First().Title() }}`,
			expected: "true Post",
		},
		{
			name:     "Last miss and hit",
			source:   `{{ nvs.ByGlob("none/*.md").Last() == nil }} {{ nvs.ByGlob("blog/*.md").Last().Title() }}`,
			expected: "true Post",
		},
		{
			name:     "LangAlternative miss",
			source:   `{{ nvs.ByPath("about.md").LangAlternative("ru") == nil }}`,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderJet(t, tt.source)
			require.NoError(t, err)
			require.Equal(t, tt.expected, out)
		})
	}
}

func TestJetAssignmentInIf(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "miss takes else",
			source:   `{{ if x := nvs.ByPath("/missing.md"); x }}{{ x.Title() }}{{ else }}none{{ end }}`,
			expected: "none",
		},
		{
			name:     "hit takes the if branch",
			source:   `{{ if x := nvs.ByPath("/about.md"); x }}{{ x.Title() }}{{ else }}none{{ end }}`,
			expected: "About",
		},
		{
			name:     "variable is visible in else",
			source:   `{{ if x := nvs.ByPath("/missing.md"); x }}found{{ else }}{{ x == nil }}{{ end }}`,
			expected: "true",
		},
		{
			name:     "condition can be any expression over the variable",
			source:   `{{ if x := nvs.ByPath("/about.md"); x != nil && x.Title() == "About" }}yes{{ end }}`,
			expected: "yes",
		},
		{
			name:     "else if chain with its own assignment",
			source:   `{{ if a := nvs.ByPath("/missing.md"); a }}a{{ else if b := nvs.ByPath("/about.md"); b }}{{ b.Title() }}{{ else }}none{{ end }}`,
			expected: "About",
		},
		{
			name:     "else if sees the outer variable",
			source:   `{{ if a := nvs.ByPath("/missing.md"); a }}a{{ else if b := nvs.ByPath("/about.md"); b && a == nil }}outer visible{{ end }}`,
			expected: "outer visible",
		},
		{
			name:     "meta value with a fallback",
			source:   `{{ if s := nvs.ByPath("/about.md").M().Get("subtitle"); s }}{{ s }}{{ else }}no subtitle{{ end }}`,
			expected: "Who we are",
		},
		{
			name:     "missing meta value falls back",
			source:   `{{ if s := nvs.ByPath("/blog/post.md").M().Get("subtitle"); s }}{{ s }}{{ else }}no subtitle{{ end }}`,
			expected: "no subtitle",
		},
		{
			name:     "query First",
			source:   `{{ if p := nvs.ByGlob("blog/*.md").First(); p }}{{ p.Title() }}{{ else }}no posts{{ end }}`,
			expected: "Post",
		},
		{
			name:     "meta string with a fallback",
			source:   `{{ if s := nvs.ByPath("/about.md").M().GetString("subtitle", ""); s != "" }}{{ s }}{{ else }}no subtitle{{ end }}`,
			expected: "Who we are",
		},
		{
			name:     "else if reuses the variable",
			source:   `{{ if x := nvs.ByPath("/missing.md"); x }}{{ x.Title() }}{{ else if x == nil }}none{{ end }}`,
			expected: "none",
		},
		{
			name:     "two-value map lookup",
			source:   `{{ m := map("k", "v") }}{{ if v, ok := m["k"]; ok }}{{ v }}{{ end }}{{ if v, ok := m["missing"]; !ok }}absent{{ end }}`,
			expected: "vabsent",
		},
		{
			name:     "assignment with = writes the outer variable",
			source:   `{{ x := "before" }}{{ if x = nvs.ByPath("/about.md"); x }}{{ end }}{{ x.Title() }}`,
			expected: "About",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderJet(t, tt.source)
			require.NoError(t, err)
			require.Equal(t, tt.expected, out)
		})
	}
}

func TestJetAssignmentInIfIsScoped(t *testing.T) {
	_, err := renderJet(t, `{{ if x := nvs.ByPath("/about.md"); x }}{{ end }}{{ x.Title() }}`)
	require.ErrorContains(t, err, `identifier "x" not available`)
}

func TestJetAssignmentInIfShadowsOuterVariable(t *testing.T) {
	out, err := renderJet(t, `{{ x := "outer" }}{{ if x := nvs.ByPath("/about.md"); x }}{{ x.Title() }}{{ end }} {{ x }}`)
	require.NoError(t, err)
	require.Equal(t, "About outer", out)
}

func TestJetRangeAssignment(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "one variable over a slice is the index",
			source:   `{{ range i := nvs.ByGlob("blog/*.md").All() }}{{ i }}{{ end }}`,
			expected: "0",
		},
		{
			name:     "two variables are index and value",
			source:   `{{ range i, p := nvs.ByGlob("blog/*.md").All() }}{{ i }}:{{ p.Title() }}{{ end }}`,
			expected: "0:Post",
		},
		{
			name:     "no variable makes the element the context",
			source:   `{{ range nvs.ByGlob("blog/*.md").All() }}{{ .Title() }}{{ end }}`,
			expected: "Post",
		},
		{
			name:     "else runs for an empty collection",
			source:   `{{ range i, p := nvs.ByGlob("none/*.md").All() }}{{ p.Title() }}{{ else }}empty{{ end }}`,
			expected: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderJet(t, tt.source)
			require.NoError(t, err)
			require.Equal(t, tt.expected, out)
		})
	}
}

func TestJetRangeAssignmentIsScoped(t *testing.T) {
	_, err := renderJet(t, `{{ range i, p := nvs.ByGlob("blog/*.md").All() }}{{ end }}{{ p.Title() }}`)
	require.ErrorContains(t, err, `identifier "p" not available`)
}

func TestJetRangeRejectsCondition(t *testing.T) {
	_, err := renderJet(t, `{{ range i, p := nvs.ByGlob("blog/*.md").All(); p }}{{ end }}`)
	require.ErrorContains(t, err, "unexpected token ';'")
}

func TestJetMethodOnMissIsARenderError(t *testing.T) {
	_, err := renderJet(t, `{{ nvs.ByPath("/missing.md").Title() }}`)
	require.ErrorContains(t, err, "nil pointer evaluating")
}

func renderJetWithNote(t *testing.T, content, source string) string {
	t.Helper()

	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{Path: "page.md", Content: []byte(content)}},
		Log:     &logger.TestLogger{},
	})
	require.NoError(t, err)

	set := jet.NewSet(jet.NewInMemLoader())

	tpl, err := set.Parse("test.jet", source)
	require.NoError(t, err)

	vars := make(jet.VarMap)
	vars.Set("note", templateviews.NewNote(pages.List[0]))

	var out bytes.Buffer
	err = tpl.Execute(&out, vars, nil)
	require.NoError(t, err)

	return out.String()
}

func TestJetPartialRendererNilChecks(t *testing.T) {
	content := "# Page\n\n## Features\n\n### Security\n\nSafe.\n"

	tests := []struct {
		name     string
		content  string
		source   string
		expected string
	}{
		{
			name:     "Section miss",
			content:  content,
			source:   `{{ s := note.PartialRenderer().Section("Missing") }}{{ if s == nil }}A{{ end }}{{ if !s }}B{{ end }}{{ if s }}C{{ end }}`,
			expected: "AB",
		},
		{
			name:     "Section hit",
			content:  content,
			source:   `{{ s := note.PartialRenderer().Section("Features") }}{{ s == nil }} {{ s.Title }}`,
			expected: "false Features",
		},
		{
			name:     "nested Section miss and hit",
			content:  content,
			source:   `{{ if f := note.PartialRenderer().Section("Features"); f }}{{ f.Section("Missing") == nil }} {{ f.Section("Security").Title }}{{ end }}`,
			expected: "true Security",
		},
		{
			name:     "FirstList miss",
			content:  content,
			source:   `{{ if l := note.PartialRenderer().FirstList(); l }}list{{ else }}no list{{ end }} {{ note.PartialRenderer().FirstList() == nil }}`,
			expected: "no list true",
		},
		{
			name:     "FirstList hit",
			content:  "- one\n- two\n",
			source:   `{{ if l := note.PartialRenderer().FirstList(); l }}{{ len(l.Items) }}{{ else }}no list{{ end }}`,
			expected: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, renderJetWithNote(t, tt.content, tt.source))
		})
	}
}
